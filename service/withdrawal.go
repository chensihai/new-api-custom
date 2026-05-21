package service

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"gorm.io/gorm"
)

var (
	ErrWithdrawalAmountMustBePositive = errors.New("提现金额必须为正数")
	ErrRebateNotAvailable             = errors.New("返利功能未启用或无权限")
	ErrWithdrawalHasDeficit           = errors.New("存在未清零欠扣，无法提现")
	ErrWithdrawalQuotaInsufficient    = errors.New("可提现额度不足")
	ErrWithdrawalRequestNotFound      = errors.New("提现申请不存在")
	ErrWithdrawalRequestAlreadyProcessed = errors.New("提现申请已处理")
	ErrWithdrawalNotCancelable        = errors.New("提现申请无法取消")
	ErrNoPermissionWithdrawal         = errors.New("无权操作此提现申请")
)

func CreateWithdrawalRequest(userId int, amount int) (*model.WithdrawalRequest, error) {
	if amount <= 0 {
		return nil, ErrWithdrawalAmountMustBePositive
	}

	user, err := model.GetUserById(userId, true)
	if err != nil || user == nil {
		return nil, ErrRebateNotAvailable
	}

	if !setting.IsRebateEnabled() || !setting.IsUserRebateVisible(user.Group) {
		return nil, ErrRebateNotAvailable
	}

	var req *model.WithdrawalRequest
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		account, err := model.GetRebateAccountForUpdate(tx, userId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrWithdrawalQuotaInsufficient
			}
			return err
		}

		if account.DeficitQuota > 0 {
			return ErrWithdrawalHasDeficit
		}

		frozenWithdrawal, err := model.SumPendingAmountByUserId(userId)
		if err != nil {
			return err
		}

		transferable := account.SettledQuota - frozenWithdrawal
		if transferable < amount {
			return ErrWithdrawalQuotaInsufficient
		}

		req = &model.WithdrawalRequest{
			UserId: userId,
			Amount: amount,
			Status: model.WithdrawalStatusPending,
		}
		if err := tx.Create(req).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	common.SysLog(fmt.Sprintf("提现申请创建成功 user_id=%d amount=%d request_id=%d", userId, amount, req.Id))
	return req, nil
}

func CancelWithdrawalRequest(userId int, requestId int) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		req, err := model.GetWithdrawalRequestForUpdate(tx, requestId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrWithdrawalRequestNotFound
			}
			return err
		}

		if req.UserId != userId {
			return ErrNoPermissionWithdrawal
		}

		if req.Status != model.WithdrawalStatusPending {
			return ErrWithdrawalNotCancelable
		}

		if err := model.UpdateWithdrawalRequestStatus(tx, requestId, model.WithdrawalStatusCancelled, 0, ""); err != nil {
			return err
		}

		common.SysLog(fmt.Sprintf("提现申请取消成功 user_id=%d request_id=%d", userId, requestId))
		return nil
	})
}

func ApproveWithdrawalRequest(adminId int, requestId int, approved bool, rejectReason string) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		req, err := model.GetWithdrawalRequestForUpdate(tx, requestId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrWithdrawalRequestNotFound
			}
			return err
		}

		if req.Status != model.WithdrawalStatusPending {
			return ErrWithdrawalRequestAlreadyProcessed
		}

		if approved {
			account, err := model.GetRebateAccountForUpdate(tx, req.UserId)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrWithdrawalQuotaInsufficient
				}
				return err
			}

			if account.SettledQuota < req.Amount {
				return ErrWithdrawalQuotaInsufficient
			}

			if err := tx.Model(account).Updates(map[string]interface{}{
				"settled_quota":             account.SettledQuota - req.Amount,
				"total_transferred_quota":   account.TotalTransferredQuota + req.Amount,
			}).Error; err != nil {
				return err
			}

			if err := model.UpdateWithdrawalRequestStatus(tx, requestId, model.WithdrawalStatusApproved, adminId, ""); err != nil {
				return err
			}

			common.SysLog(fmt.Sprintf("提现申请审批通过 admin_id=%d request_id=%d user_id=%d amount=%d", adminId, requestId, req.UserId, req.Amount))
		} else {
			if err := model.UpdateWithdrawalRequestStatus(tx, requestId, model.WithdrawalStatusRejected, adminId, rejectReason); err != nil {
				return err
			}

			common.SysLog(fmt.Sprintf("提现申请审批拒绝 admin_id=%d request_id=%d user_id=%d reason=%s", adminId, requestId, req.UserId, rejectReason))
		}

		return nil
	})
}
