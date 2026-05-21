package model

const (
	RebateStatusPending    = "pending"
	RebateStatusSettled    = "settled"
	RebateStatusFrozen     = "frozen"
	RebateStatusClawedBack = "clawed_back"
	RebateStatusTransferred = "transferred"
)

const (
	CappedReasonNone           = "none"
	CappedReasonPartiallyCapped = "partially_capped"
	CappedReasonFullyCapped    = "fully_capped"
)

type RebateRecord struct {
	Id               int    `json:"id" gorm:"primaryKey"`
	InviterId        int    `json:"inviter_id" gorm:"type:int;index;column:inviter_id"`
	InviteeId        int    `json:"invitee_id" gorm:"type:int;column:invitee_id"`
	RechargeLogId    int    `json:"recharge_log_id" gorm:"type:int;index;column:recharge_log_id"`
	RechargeQuota    int    `json:"recharge_quota" gorm:"type:int;default:0;column:recharge_quota"`
	RebateQuota      int    `json:"rebate_quota" gorm:"type:int;default:0;column:rebate_quota"`
	RebateRate       float64 `json:"rebate_rate" gorm:"type:decimal(5,2);default:0;column:rebate_rate"`
	Status           string `json:"status" gorm:"type:varchar(20);default:'pending';column:status"`
	OriginalStatus   string `json:"original_status" gorm:"type:varchar(20);default:'';column:original_status"`
	ExpectedSettleAt int64  `json:"expected_settle_at" gorm:"type:bigint;default:0;column:expected_settle_at"`
	CappedReason     string `json:"capped_reason" gorm:"type:varchar(30);default:'none';column:capped_reason"`
	DeficitQuota     int    `json:"deficit_quota" gorm:"type:int;default:0;column:deficit_quota"`
	CreatedAt        int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt        int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

func (RebateRecord) TableName() string {
	return "rebate_records"
}

func GetPendingRecordsBefore(now int64, batch int) ([]RebateRecord, error) {
	var records []RebateRecord
	err := DB.Where("status = ? AND expected_settle_at <= ? AND expected_settle_at > 0", RebateStatusPending, now).
		Order("expected_settle_at asc").
		Limit(batch).
		Find(&records).Error
	return records, err
}

func GetRecordsByRechargeLogId(topUpId int) ([]RebateRecord, error) {
	var records []RebateRecord
	err := DB.Where("recharge_log_id = ?", topUpId).Find(&records).Error
	return records, err
}

func GetRecordsByInviterIdAndStatus(inviterId int, status string, page int, pageSize int) ([]RebateRecord, int64, error) {
	var records []RebateRecord
	var total int64
	query := DB.Where("inviter_id = ? AND status = ?", inviterId, status)
	if err := query.Model(&RebateRecord{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func GetSettledRecordsForTransfer(inviterId int, limit int, offset int) ([]RebateRecord, error) {
	var records []RebateRecord
	err := DB.Where("inviter_id = ? AND status = ?", inviterId, RebateStatusSettled).
		Order("created_at asc").
		Offset(offset).
		Limit(limit).
		Find(&records).Error
	return records, err
}

func GetRecordByRechargeLogId(topUpId int) (*RebateRecord, error) {
	var record RebateRecord
	err := DB.Where("recharge_log_id = ?", topUpId).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func GetRecordsByRechargeLogIdAndStatuses(topUpId int, statuses []string) ([]RebateRecord, error) {
	var records []RebateRecord
	err := DB.Where("recharge_log_id = ? AND status IN ?", topUpId, statuses).Find(&records).Error
	return records, err
}

func GetFrozenRecordsByRechargeLogId(topUpId int) ([]RebateRecord, error) {
	var records []RebateRecord
	err := DB.Where("recharge_log_id = ? AND status = ?", topUpId, RebateStatusFrozen).Find(&records).Error
	return records, err
}
