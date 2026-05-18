package service

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"gorm.io/gorm"
)

func TriggerRebateOnRecharge(userId int, rechargeQuota int, topUpId int) {
	if !setting.IsRebateEnabled() {
		return
	}

	user, err := model.GetUserById(userId, true)
	if err != nil || user == nil {
		return
	}

	if user.InviterId == 0 || user.InviterId == userId {
		return
	}

	inviter, err := model.GetUserById(user.InviterId, true)
	if err != nil || inviter == nil {
		return
	}

	if !setting.IsUserRebateVisible(inviter.Group) {
		return
	}

	if inviter.RebateRate == 0 {
		return
	}

	existing, err := model.GetRecordByRechargeLogId(topUpId)
	if err == nil && existing != nil {
		return
	}

	rebateQuota := (rechargeQuota * inviter.RebateRate) / 10000
	if rebateQuota == 0 {
		return
	}

	settlementPeriod := setting.GetSettlementPeriod()
	expectedSettleAt := time.Now().Add(time.Duration(settlementPeriod) * 24 * time.Hour).Unix()

	var actualRebateQuota int
	var cappedReason string

	err = model.DB.Transaction(func(tx *gorm.DB) error {
		actualRebateQuota = rebateQuota
		cappedReason = model.CappedReasonNone

		if inviter.RebateCap > 0 {
			capProgress, err := model.GetOrCreateCapProgressForUpdate(tx, inviter.Id, userId)
			if err != nil {
				return fmt.Errorf("获取封顶进度失败: %w", err)
			}
			remaining := inviter.RebateCap - capProgress.TotalRebateQuota
			if remaining <= 0 {
				cappedReason = model.CappedReasonFullyCapped
				actualRebateQuota = 0
			} else if actualRebateQuota > remaining {
				cappedReason = model.CappedReasonPartiallyCapped
				actualRebateQuota = remaining
			}
		}

		if actualRebateQuota == 0 {
			if cappedReason == model.CappedReasonFullyCapped {
				logger.LogInfo(nil, fmt.Sprintf("返利计算 全额封顶 inviter_id=%d invitee_id=%d top_up_id=%d cap=%d", inviter.Id, userId, topUpId, inviter.RebateCap))
			}
			return nil
		}

		record := &model.RebateRecord{
			InviterId:        inviter.Id,
			InviteeId:        userId,
			RechargeLogId:    topUpId,
			RechargeQuota:    rechargeQuota,
			RebateQuota:      actualRebateQuota,
			RebateRate:       inviter.RebateRate,
			Status:           model.RebateStatusPending,
			OriginalStatus:   "",
			ExpectedSettleAt: expectedSettleAt,
			CappedReason:     cappedReason,
		}
		if err := tx.Create(record).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.RebateAccount{}).Where("user_id = ?", inviter.Id).
			Update("pending_quota", gorm.Expr("pending_quota + ?", actualRebateQuota)).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				newAccount := &model.RebateAccount{
					UserId:       inviter.Id,
					PendingQuota: actualRebateQuota,
				}
				if err := tx.Create(newAccount).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}

		if inviter.RebateCap > 0 {
			if err := tx.Model(&model.RebateCapProgress{}).Where("inviter_id = ? AND invitee_id = ?", inviter.Id, userId).
				Update("total_rebate_quota", gorm.Expr("total_rebate_quota + ?", actualRebateQuota)).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		logger.LogError(nil, fmt.Sprintf("返利计算 事务失败 inviter_id=%d invitee_id=%d top_up_id=%d error=%q", inviter.Id, userId, topUpId, err.Error()))
		return
	}

	if actualRebateQuota > 0 {
		logger.LogInfo(nil, fmt.Sprintf("返利计算成功 inviter_id=%d invitee_id=%d top_up_id=%d recharge_quota=%d rebate_quota=%d rate=%d cap_reason=%s", inviter.Id, userId, topUpId, rechargeQuota, actualRebateQuota, inviter.RebateRate, cappedReason))
	}
}

func MaskUsername(name string) string {
	runes := []rune(name)
	l := len(runes)
	if l <= 4 {
		if l <= 1 {
			return name
		}
		return string(runes[0]) + "***"
	}
	return string(runes[:2]) + "***" + string(runes[l-2:])
}
