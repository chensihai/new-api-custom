package service

import (
	"fmt"

	"github.com/QuantumNous/new-api/model"

	"gorm.io/gorm"
)

func settleRebateRecord(record *model.RebateRecord) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var lockedRecord model.RebateRecord
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&lockedRecord, record.Id).Error; err != nil {
			return err
		}
		if lockedRecord.Status != model.RebateStatusPending {
			return nil
		}

		deficitTotal, err := model.GetActiveDeficitTotalByInviterId(lockedRecord.InviterId)
		if err != nil {
			return fmt.Errorf("查询欠扣总额失败: %w", err)
		}

		remaining := lockedRecord.RebateQuota
		deficitDeducted := 0

		if deficitTotal > 0 {
			deficits, err := model.GetActiveDeficitsByInviterId(lockedRecord.InviterId)
			if err != nil {
				return fmt.Errorf("查询欠扣记录失败: %w", err)
			}

			leftToDeduct := remaining
			if leftToDeduct > deficitTotal {
				leftToDeduct = deficitTotal
			}

			for _, deficit := range deficits {
				if leftToDeduct <= 0 {
					break
				}
				deduct := deficit.RemainingDeficitQuota
				if deduct > leftToDeduct {
					deduct = leftToDeduct
				}
				newRemaining := deficit.RemainingDeficitQuota - deduct
				status := model.DeficitStatusActive
				if newRemaining <= 0 {
					status = model.DeficitStatusCleared
					newRemaining = 0
				}
				if err := tx.Model(&deficit).Updates(map[string]interface{}{
					"remaining_deficit_quota": newRemaining,
					"status":                  status,
				}).Error; err != nil {
					return fmt.Errorf("更新欠扣记录失败 deficit_id=%d: %w", deficit.Id, err)
				}
				leftToDeduct -= deduct
				deficitDeducted += deduct
			}
			remaining -= deficitDeducted
		}

		if err := tx.Model(&lockedRecord).Updates(map[string]interface{}{
			"status":       model.RebateStatusSettled,
			"deficit_quota": deficitDeducted,
		}).Error; err != nil {
			return err
		}

		account, err := model.GetRebateAccountForUpdate(tx, lockedRecord.InviterId)
		if err != nil {
			return err
		}

		updates := map[string]interface{}{
			"pending_quota": account.PendingQuota - lockedRecord.RebateQuota,
		}
		if remaining > 0 {
			updates["settled_quota"] = account.SettledQuota + remaining
			updates["total_settled_quota"] = account.TotalSettledQuota + remaining
		}
		if deficitDeducted > 0 {
			updates["deficit_quota"] = account.DeficitQuota - deficitDeducted
		}

		return tx.Model(account).Updates(updates).Error
	})
}
