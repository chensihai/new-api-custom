package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func CreateWithdrawal(c *gin.Context) {
	userId := c.GetInt("id")

	var req struct {
		Amount int `json:"amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	result, err := service.CreateWithdrawalRequest(userId, req.Amount)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "提现申请已提交", "data": result})
}

func GetWithdrawalRequests(c *gin.Context) {
	userId := c.GetInt("id")
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	requests, total, err := model.GetWithdrawalRequestsByUserId(userId, status, page, pageSize)
	if err != nil {
		common.ApiErrorMsg(c, "查询失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"requests": requests, "total": total}})
}

func CancelWithdrawal(c *gin.Context) {
	userId := c.GetInt("id")
	requestId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	if err := service.CancelWithdrawalRequest(userId, requestId); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "提现申请已取消"})
}

func AdminGetWithdrawalRequests(c *gin.Context) {
	userId, _ := strconv.Atoi(c.DefaultQuery("user_id", "0"))
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	requests, total, err := model.GetWithdrawalRequestsAll(userId, status, page, pageSize)
	if err != nil {
		common.ApiErrorMsg(c, "查询失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"requests": requests, "total": total}})
}

func AdminApproveWithdrawal(c *gin.Context) {
	adminId := c.GetInt("id")
	requestId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	var req struct {
		Approved     bool   `json:"approved"`
		RejectReason string `json:"reject_reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	if err := service.ApproveWithdrawalRequest(adminId, requestId, req.Approved, req.RejectReason); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	msg := "提现申请已拒绝"
	if req.Approved {
		msg = "提现申请已通过"
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": msg})
}
