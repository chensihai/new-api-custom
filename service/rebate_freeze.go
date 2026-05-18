package service

import (
	"fmt"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"gorm.io/gorm"
)

func TriggerRebateOnRefund(inviteeId int, topUpId int) {
	records, err := model.GetRecordsByRechargeLogIdAndStatuses(topUpId, []string{model.RebateStatusPending, model.RebateStatusSettled})
	if err != nil {
		logger.LogError(nil, fmt.Sprintf("返利冻结 查询记录失败 top_up_id=%d error=%q", topUpId, err.Error()))
		return
	}
	if len(records) == 0 {
		return
	}

	for _, record := range records {
		if err := freezeRebateRecord(&record); err != nil {
			logger.LogError(nil, fmt.Sprintf("返利冻结 冻结记录失败 record_id=%d error=%q", record.Id, err.Error()))
			continue
		}
		logger.LogInfo(nil, fmt.Sprintf("返利冻结成功 record_id=%d inviter_id=%d rebate_quota=%d prev_status=%s", record.Id, record.InviterId, record.RebateQuota, record.Status))
	}
}

func freezeRebateRecord(record *model.RebateRecord) error {
	originalStatus := record.Status

	return model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(record).Updates(map[string]interface{}{
			"status":          model.RebateStatusFrozen,
			"original_status": originalStatus,
		}).Error; err != nil {
			return err
		}

		account, err := model.GetRebateAccountForUpdate(tx, record.InviterId)
		if err != nil {
			return err
		}

		updates := map[string]interface{}{
			"frozen_quota": account.FrozenQuota + record.RebateQuota,
		}
		if originalStatus == model.RebateStatusPending {
			updates["pending_quota"] = account.PendingQuota - record.RebateQuota
		} else if originalStatus == model.RebateStatusSettled {
			updates["settled_quota"] = account.SettledQuota - record.RebateQuota
		}

		return tx.Model(account).Updates(updates).Error
	})
}

func TriggerRebateOnRefundConfirm(topUpId int, refundLogId int) {
	records, err := model.GetRecordsByRechargeLogIdAndStatuses(topUpId, []string{model.RebateStatusFrozen, model.RebateStatusTransferred})
	if err != nil {
		logger.LogError(nil, fmt.Sprintf("返利扣回 查询记录失败 top_up_id=%d error=%q", topUpId, err.Error()))
		return
	}
	if len(records) == 0 {
		return
	}

	for _, record := range records {
		if record.Status == model.RebateStatusFrozen {
			if err := clawBackFrozenRecord(&record); err != nil {
				logger.LogError(nil, fmt.Sprintf("返利扣回 扣回冻结记录失败 record_id=%d error=%q", record.Id, err.Error()))
				continue
			}
			logger.LogInfo(nil, fmt.Sprintf("返利扣回成功(冻结→扣回) record_id=%d inviter_id=%d rebate_quota=%d", record.Id, record.InviterId, record.RebateQuota))
		} else if record.Status == model.RebateStatusTransferred {
			if err := createDeficitFromTransferredRecord(&record, refundLogId); err != nil {
				logger.LogError(nil, fmt.Sprintf("返利欠扣 创建欠扣记录失败 record_id=%d error=%q", record.Id, err.Error()))
				continue
			}
			logger.LogInfo(nil, fmt.Sprintf("返利欠扣成功(已划转→欠扣) record_id=%d inviter_id=%d rebate_quota=%d", record.Id, record.InviterId, record.RebateQuota))
		}
	}
}

func clawBackFrozenRecord(record *model.RebateRecord) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(record).Updates(map[string]interface{}{
			"status": model.RebateStatusClawedBack,
		}).Error; err != nil {
			return err
		}

		account, err := model.GetRebateAccountForUpdate(tx, record.InviterId)
		if err != nil {
			return err
		}

		return tx.Model(account).Updates(map[string]interface{}{
			"frozen_quota": account.FrozenQuota - record.RebateQuota,
		}).Error
	})
}

func createDeficitFromTransferredRecord(record *model.RebateRecord, refundLogId int) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		deficit := &model.RebateDeficit{
			InviterId:             record.InviterId,
			RebateRecordId:        record.Id,
			RefundLogId:           refundLogId,
			TotalDeficitQuota:     record.RebateQuota,
			RemainingDeficitQuota: record.RebateQuota,
			Status:                model.DeficitStatusActive,
		}
		if err := tx.Create(deficit).Error; err != nil {
			return err
		}

		if err := tx.Model(record).Updates(map[string]interface{}{
			"deficit_quota": record.RebateQuota,
		}).Error; err != nil {
			return err
		}

		account, err := model.GetRebateAccountForUpdate(tx, record.InviterId)
		if err != nil {
			return err
		}

		return tx.Model(account).Updates(map[string]interface{}{
			"deficit_quota": account.DeficitQuota + record.RebateQuota,
		}).Error
	})
}

func TriggerRebateOnRefundCancel(topUpId int) {
	records, err := model.GetFrozenRecordsByRechargeLogId(topUpId)
	if err != nil {
		logger.LogError(nil, fmt.Sprintf("返利解冻 查询冻结记录失败 top_up_id=%d error=%q", topUpId, err.Error()))
		return
	}
	if len(records) == 0 {
		return
	}

	for _, record := range records {
		if err := unfreezeRebateRecord(&record); err != nil {
			logger.LogError(nil, fmt.Sprintf("返利解冻 解冻记录失败 record_id=%d error=%q", record.Id, err.Error()))
			continue
		}
		logger.LogInfo(nil, fmt.Sprintf("返利解冻成功 record_id=%d inviter_id=%d rebate_quota=%d restored_status=%s", record.Id, record.InviterId, record.RebateQuota, record.OriginalStatus))
	}
}

func unfreezeRebateRecord(record *model.RebateRecord) error {
	originalStatus := record.OriginalStatus
	if originalStatus == "" {
		originalStatus = model.RebateStatusPending
	}

	return model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(record).Updates(map[string]interface{}{
			"status":          originalStatus,
			"original_status": "",
		}).Error; err != nil {
			return err
		}

		account, err := model.GetRebateAccountForUpdate(tx, record.InviterId)
		if err != nil {
			return err
		}

		updates := map[string]interface{}{
			"frozen_quota": account.FrozenQuota - record.RebateQuota,
		}
		if originalStatus == model.RebateStatusPending {
			updates["pending_quota"] = account.PendingQuota + record.RebateQuota
		} else if originalStatus == model.RebateStatusSettled {
			updates["settled_quota"] = account.SettledQuota + record.RebateQuota
		}

		return tx.Model(account).Updates(updates).Error
	})
}
