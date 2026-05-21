package service

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"gorm.io/gorm"
)

var (
	ErrRebateQuotaMustBePositive = errors.New("划转金额必须为正数")
	ErrRebateHasDeficit          = errors.New("存在未清零欠扣，无法划转")
	ErrRebateQuotaInsufficient   = errors.New("可划转返利额度不足")
)

func TransferRebate(userId int, quota int) error {
	if quota <= 0 {
		return ErrRebateQuotaMustBePositive
	}

	return model.DB.Transaction(func(tx *gorm.DB) error {
		account, err := model.GetRebateAccountForUpdate(tx, userId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRebateQuotaInsufficient
			}
			return err
		}

		if account.DeficitQuota > 0 {
			return ErrRebateHasDeficit
		}

		if account.SettledQuota < quota {
			return ErrRebateQuotaInsufficient
		}

		remaining := quota
		var offset int
		const batchSize = 100

		for remaining > 0 {
			records, err := model.GetSettledRecordsForTransfer(userId, batchSize, offset)
			if err != nil {
				return err
			}
			if len(records) == 0 {
				break
			}

			for _, record := range records {
				if remaining <= 0 {
					break
				}
				if record.RebateQuota <= remaining {
					if err := tx.Model(&record).Update("status", model.RebateStatusTransferred).Error; err != nil {
						return err
					}
					remaining -= record.RebateQuota
				} else {
					splitRecord := &model.RebateRecord{
						InviterId:        record.InviterId,
						InviteeId:        record.InviteeId,
						RechargeLogId:    0,
						RechargeQuota:    0,
						RebateQuota:      record.RebateQuota - remaining,
						RebateRate:       record.RebateRate,
						Status:           model.RebateStatusSettled,
						ExpectedSettleAt: record.ExpectedSettleAt,
						CappedReason:     record.CappedReason,
					}
					if err := tx.Create(splitRecord).Error; err != nil {
						return err
					}
					if err := tx.Model(&record).Updates(map[string]interface{}{
						"rebate_quota": remaining,
						"status":       model.RebateStatusTransferred,
					}).Error; err != nil {
						return err
					}
					remaining = 0
				}
			}

			if len(records) < batchSize {
				break
			}
			offset += batchSize
		}

		if remaining > 0 {
			return ErrRebateQuotaInsufficient
		}

		if err := tx.Model(account).Updates(map[string]interface{}{
			"settled_quota":            account.SettledQuota - quota,
			"total_transferred_quota": account.TotalTransferredQuota + quota,
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.User{}).Where("id = ?", userId).Update("quota", gorm.Expr("quota + ?", quota)).Error; err != nil {
			return fmt.Errorf("增加用户余额失败: %w", err)
		}

		common.SysLog(fmt.Sprintf("返利划转成功 user_id=%d quota=%d", userId, quota))
		return nil
	})
}
