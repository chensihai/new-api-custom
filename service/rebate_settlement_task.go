package service

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
)

const rebateSettlementBatchSize = 500

func StartRebateSettlementTask() {
	time.Sleep(30 * time.Second)
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C
		if !setting.IsRebateEnabled() {
			continue
		}
		if common.UsingPostgreSQL {
			runRebateSettlementOnce()
		} else {
			runRebateSettlementOnce()
		}
	}
}

func runRebateSettlementOnce() {
	now := time.Now().Unix()
	for {
		records, err := model.GetPendingRecordsBefore(now, rebateSettlementBatchSize)
		if err != nil {
			logger.LogError(nil, fmt.Sprintf("返利结算 查询待结算记录失败 error=%q", err.Error()))
			return
		}
		if len(records) == 0 {
			return
		}

		for _, record := range records {
			if err := settleRebateRecord(&record); err != nil {
				logger.LogError(nil, fmt.Sprintf("返利结算 结算记录失败 record_id=%d error=%q", record.Id, err.Error()))
				continue
			}
			logger.LogInfo(nil, fmt.Sprintf("返利结算成功 record_id=%d inviter_id=%d rebate_quota=%d", record.Id, record.InviterId, record.RebateQuota))
		}

		if len(records) < rebateSettlementBatchSize {
			return
		}
	}
}
