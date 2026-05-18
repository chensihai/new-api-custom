package model

import (
	"github.com/QuantumNous/new-api/common"
)

const (
	DeficitStatusActive  = "active"
	DeficitStatusCleared = "cleared"
)

type RebateDeficit struct {
	Id                  int    `json:"id" gorm:"primaryKey"`
	InviterId           int    `json:"inviter_id" gorm:"type:int;index;column:inviter_id"`
	RebateRecordId      int    `json:"rebate_record_id" gorm:"type:int;column:rebate_record_id"`
	RefundLogId         int    `json:"refund_log_id" gorm:"type:int;default:0;column:refund_log_id"`
	TotalDeficitQuota   int    `json:"total_deficit_quota" gorm:"type:int;default:0;column:total_deficit_quota"`
	RemainingDeficitQuota int  `json:"remaining_deficit_quota" gorm:"type:int;default:0;column:remaining_deficit_quota"`
	Status              string `json:"status" gorm:"type:varchar(20);default:'active';column:status"`
	CreatedAt           int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt           int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

func (RebateDeficit) TableName() string {
	return "rebate_deficits"
}

func GetActiveDeficitsByInviterId(inviterId int) ([]RebateDeficit, error) {
	var deficits []RebateDeficit
	err := DB.Where("inviter_id = ? AND status = ?", inviterId, DeficitStatusActive).
		Order("created_at asc").
		Find(&deficits).Error
	return deficits, err
}

func GetActiveDeficitTotalByInviterId(inviterId int) (int, error) {
	var total int
	refCol := "`remaining_deficit_quota`"
	if common.UsingPostgreSQL {
		refCol = `"remaining_deficit_quota"`
	}
	err := DB.Model(&RebateDeficit{}).
		Where("inviter_id = ? AND status = ?", inviterId, DeficitStatusActive).
		Select("COALESCE(SUM(" + refCol + "), 0)").
		Scan(&total).Error
	return total, err
}

func GetDeficitsByInviterId(inviterId int, page int, pageSize int) ([]RebateDeficit, int64, error) {
	var deficits []RebateDeficit
	var total int64
	query := DB.Where("inviter_id = ?", inviterId)
	if err := query.Model(&RebateDeficit{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&deficits).Error; err != nil {
		return nil, 0, err
	}
	return deficits, total, nil
}
