package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
)

func checkRebateVisibility(c *gin.Context) bool {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "未登录")
		return false
	}
	user, err := model.GetUserById(userId, true)
	if err != nil || user == nil {
		common.ApiErrorMsg(c, "用户不存在")
		return false
	}
	if !setting.IsUserRebateVisible(user.Group) {
		common.ApiErrorMsg(c, "返利功能不可用")
		return false
	}
	return true
}

func GetRebateVisibility(c *gin.Context) {
	userId := c.GetInt("id")
	visible := false
	if userId > 0 {
		user, err := model.GetUserById(userId, true)
		if err == nil && user != nil {
			visible = setting.IsUserRebateVisible(user.Group)
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"visible": visible}})
}

func GetRebateSummary(c *gin.Context) {
	if !checkRebateVisibility(c) {
		return
	}
	userId := c.GetInt("id")
	account, err := model.GetOrCreateRebateAccount(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"pending_quota":            account.PendingQuota,
			"settled_quota":            account.SettledQuota,
			"frozen_quota":             account.FrozenQuota,
			"deficit_quota":            account.DeficitQuota,
			"total_settled_quota":      account.TotalSettledQuota,
			"total_transferred_quota":  account.TotalTransferredQuota,
			"transferable_quota":       account.SettledQuota,
		},
	})
}

func GetRebateRecords(c *gin.Context) {
	if !checkRebateVisibility(c) {
		return
	}
	userId := c.GetInt("id")
	status := c.DefaultQuery("status", model.RebateStatusPending)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	records, total, err := model.GetRecordsByInviterIdAndStatus(userId, status, page, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	items := make([]gin.H, 0, len(records))
	for _, r := range records {
		invitee, _ := model.GetUserById(r.InviteeId, true)
		inviteeName := ""
		if invitee != nil {
			inviteeName = service.MaskUsername(invitee.DisplayName)
			if inviteeName == "" {
				inviteeName = service.MaskUsername(invitee.Username)
			}
		}
		items = append(items, gin.H{
			"id":                 r.Id,
			"invitee_name":       inviteeName,
			"recharge_quota":     r.RechargeQuota,
			"rebate_quota":       r.RebateQuota,
			"rebate_rate":        r.RebateRate,
			"status":             r.Status,
			"expected_settle_at": r.ExpectedSettleAt,
			"capped_reason":      r.CappedReason,
			"created_at":         r.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    items,
		"total":   total,
		"page":    page,
		"page_size": pageSize,
	})
}

func GetRebateDeficits(c *gin.Context) {
	if !checkRebateVisibility(c) {
		return
	}
	userId := c.GetInt("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	deficits, total, err := model.GetDeficitsByInviterId(userId, page, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	items := make([]gin.H, 0, len(deficits))
	for _, d := range deficits {
		items = append(items, gin.H{
			"id":                    d.Id,
			"rebate_record_id":      d.RebateRecordId,
			"refund_log_id":         d.RefundLogId,
			"total_deficit_quota":   d.TotalDeficitQuota,
			"remaining_deficit_quota": d.RemainingDeficitQuota,
			"status":                d.Status,
			"created_at":            d.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    items,
		"total":   total,
		"page":    page,
		"page_size": pageSize,
	})
}

func GetRebateInviteeProgress(c *gin.Context) {
	if !checkRebateVisibility(c) {
		return
	}
	userId := c.GetInt("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	progresses, total, err := model.GetCapProgressByInviterId(userId, page, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	items := make([]gin.H, 0, len(progresses))
	for _, p := range progresses {
		invitee, _ := model.GetUserById(p.InviteeId, true)
		inviteeName := ""
		if invitee != nil {
			inviteeName = service.MaskUsername(invitee.DisplayName)
			if inviteeName == "" {
				inviteeName = service.MaskUsername(invitee.Username)
			}
		}
		items = append(items, gin.H{
			"invitee_name":       inviteeName,
			"total_rebate_quota": p.TotalRebateQuota,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    items,
		"total":   total,
		"page":    page,
		"page_size": pageSize,
	})
}

func TransferRebate(c *gin.Context) {
	if !checkRebateVisibility(c) {
		return
	}
	userId := c.GetInt("id")

	var req struct {
		Quota int `json:"quota"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	if err := service.TransferRebate(userId, req.Quota); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "划转成功"})
}

func GetRebateSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"rebate_enabled":        setting.IsRebateEnabled(),
			"settlement_period":     setting.GetSettlementPeriod(),
			"rebate_visible_groups": setting.GetRebateVisibleGroups(),
		},
	})
}

func UpdateRebateSettings(c *gin.Context) {
	var req struct {
		RebateEnabled       *bool  `json:"rebate_enabled"`
		SettlementPeriod    *int   `json:"settlement_period"`
		RebateVisibleGroups *string `json:"rebate_visible_groups"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	enabled := setting.IsRebateEnabled()
	period := setting.GetSettlementPeriod()
	groups := setting.GetRebateVisibleGroups()

	if req.RebateEnabled != nil {
		enabled = *req.RebateEnabled
	}
	if req.SettlementPeriod != nil {
		if *req.SettlementPeriod < 1 {
			common.ApiErrorMsg(c, "结算期必须为正整数且最小1天")
			return
		}
		period = *req.SettlementPeriod
	}
	if req.RebateVisibleGroups != nil {
		groups = *req.RebateVisibleGroups
	}

	if req.RebateEnabled != nil {
		_ = model.UpdateOption("rebate_setting.enabled", strconv.FormatBool(enabled))
	}
	if req.SettlementPeriod != nil {
		_ = model.UpdateOption("rebate_setting.settlement_period", strconv.Itoa(period))
	}
	if req.RebateVisibleGroups != nil {
		_ = model.UpdateOption("rebate_setting.visible_groups", groups)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "更新成功"})
}

func GetRebateStatistics(c *gin.Context) {
	var stats struct {
		TotalPendingQuota     int64
		TotalSettledQuota     int64
		TotalFrozenQuota      int64
		TotalDeficitQuota     int64
		TotalTransferredQuota int64
		TotalRecords          int64
		ActiveDeficits        int64
	}
	model.DB.Model(&model.RebateAccount{}).Select("COALESCE(SUM(pending_quota),0)").Scan(&stats.TotalPendingQuota)
	model.DB.Model(&model.RebateAccount{}).Select("COALESCE(SUM(settled_quota),0)").Scan(&stats.TotalSettledQuota)
	model.DB.Model(&model.RebateAccount{}).Select("COALESCE(SUM(frozen_quota),0)").Scan(&stats.TotalFrozenQuota)
	model.DB.Model(&model.RebateAccount{}).Select("COALESCE(SUM(deficit_quota),0)").Scan(&stats.TotalDeficitQuota)
	model.DB.Model(&model.RebateAccount{}).Select("COALESCE(SUM(total_transferred_quota),0)").Scan(&stats.TotalTransferredQuota)
	model.DB.Model(&model.RebateRecord{}).Count(&stats.TotalRecords)
	model.DB.Model(&model.RebateDeficit{}).Where("status = ?", model.DeficitStatusActive).Count(&stats.ActiveDeficits)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

func AdminGetUserRebateSummary(c *gin.Context) {
	userIdStr := c.Param("id")
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	account, err := model.GetOrCreateRebateAccount(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": account})
}

func AdminGetUserRebateRecords(c *gin.Context) {
	userIdStr := c.Param("id")
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	status := c.DefaultQuery("status", model.RebateStatusPending)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	records, total, err := model.GetRecordsByInviterIdAndStatus(userId, status, page, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": records, "total": total, "page": page, "page_size": pageSize})
}

func AdminGetUserRebateDeficits(c *gin.Context) {
	userIdStr := c.Param("id")
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	deficits, total, err := model.GetDeficitsByInviterId(userId, page, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": deficits, "total": total, "page": page, "page_size": pageSize})
}
