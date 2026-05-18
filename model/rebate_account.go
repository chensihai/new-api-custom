package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type RebateAccount struct {
	Id                   int    `json:"id" gorm:"primaryKey"`
	UserId               int    `json:"user_id" gorm:"type:int;uniqueIndex;column:user_id"`
	PendingQuota         int    `json:"pending_quota" gorm:"type:int;default:0;column:pending_quota"`
	SettledQuota         int    `json:"settled_quota" gorm:"type:int;default:0;column:settled_quota"`
	FrozenQuota          int    `json:"frozen_quota" gorm:"type:int;default:0;column:frozen_quota"`
	DeficitQuota         int    `json:"deficit_quota" gorm:"type:int;default:0;column:deficit_quota"`
	TotalSettledQuota    int    `json:"total_settled_quota" gorm:"type:int;default:0;column:total_settled_quota"`
	TotalTransferredQuota int   `json:"total_transferred_quota" gorm:"type:int;default:0;column:total_transferred_quota"`
	UpdatedAt            int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

func (RebateAccount) TableName() string {
	return "rebate_accounts"
}

func GetOrCreateRebateAccount(userId int) (*RebateAccount, error) {
	var account RebateAccount
	err := DB.Where("user_id = ?", userId).First(&account).Error
	if err == nil {
		return &account, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	account = RebateAccount{
		UserId: userId,
	}
	if err := DB.Create(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func UpdateRebateAccountByUserId(userId int, updates map[string]interface{}) error {
	refCol := "`user_id`"
	if common.UsingPostgreSQL {
		refCol = `"user_id"`
	}
	return DB.Model(&RebateAccount{}).Where(refCol+" = ?", userId).Updates(updates).Error
}

func GetRebateAccountForUpdate(tx *gorm.DB, userId int) (*RebateAccount, error) {
	var account RebateAccount
	refCol := "`user_id`"
	if common.UsingPostgreSQL {
		refCol = `"user_id"`
	}
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", userId).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}
