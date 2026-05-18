package model

import (
	"errors"

	"gorm.io/gorm"
)

const (
	RebateCapNoLimit = 0
)

type RebateCapProgress struct {
	Id               int   `json:"id" gorm:"primaryKey"`
	InviterId        int   `json:"inviter_id" gorm:"type:int;uniqueIndex:idx_inviter_invitee;column:inviter_id"`
	InviteeId        int   `json:"invitee_id" gorm:"type:int;uniqueIndex:idx_inviter_invitee;column:invitee_id"`
	TotalRebateQuota int   `json:"total_rebate_quota" gorm:"type:int;default:0;column:total_rebate_quota"`
	UpdatedAt        int64 `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

func (RebateCapProgress) TableName() string {
	return "rebate_cap_progresses"
}

func GetOrCreateCapProgress(inviterId int, inviteeId int) (*RebateCapProgress, error) {
	var progress RebateCapProgress
	err := DB.Where("inviter_id = ? AND invitee_id = ?", inviterId, inviteeId).First(&progress).Error
	if err == nil {
		return &progress, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	progress = RebateCapProgress{
		InviterId:        inviterId,
		InviteeId:        inviteeId,
		TotalRebateQuota: 0,
	}
	if err := DB.Create(&progress).Error; err != nil {
		return nil, err
	}
	return &progress, nil
}

func GetOrCreateCapProgressForUpdate(tx *gorm.DB, inviterId int, inviteeId int) (*RebateCapProgress, error) {
	var progress RebateCapProgress
	err := tx.Set("gorm:query_option", "FOR UPDATE").Where("inviter_id = ? AND invitee_id = ?", inviterId, inviteeId).First(&progress).Error
	if err == nil {
		return &progress, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	progress = RebateCapProgress{
		InviterId:        inviterId,
		InviteeId:        inviteeId,
		TotalRebateQuota: 0,
	}
	if err := tx.Create(&progress).Error; err != nil {
		return nil, err
	}
	return &progress, nil
}

func GetCapProgressByInviterId(inviterId int, page int, pageSize int) ([]RebateCapProgress, int64, error) {
	var progresses []RebateCapProgress
	var total int64
	query := DB.Where("inviter_id = ? AND total_rebate_quota > 0", inviterId)
	if err := query.Model(&RebateCapProgress{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := query.Order("updated_at desc").Offset(offset).Limit(pageSize).Find(&progresses).Error; err != nil {
		return nil, 0, err
	}
	return progresses, total, nil
}
