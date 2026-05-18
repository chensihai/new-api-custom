package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
)

type SubscriptionWechatPayRequest struct {
	PlanId        int    `json:"plan_id"`
	PaymentMethod string `json:"payment_method"`
}

func SubscriptionRequestWechatPay(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	var req SubscriptionWechatPayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	if req.PaymentMethod != model.PaymentMethodWechatNative && req.PaymentMethod != model.PaymentMethodWechatH5 {
		common.ApiErrorMsg(c, "支付方式不存在")
		return
	}

	if !isWechatTopUpEnabled() {
		common.ApiErrorMsg(c, "微信支付未启用")
		return
	}

	plan, err := model.GetSubscriptionPlanById(req.PlanId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !plan.Enabled {
		common.ApiErrorMsg(c, "套餐未启用")
		return
	}
	if plan.PriceAmount < 0.01 {
		common.ApiErrorMsg(c, "套餐金额过低")
		return
	}

	userId := c.GetInt("id")
	if plan.MaxPurchasePerUser > 0 {
		count, err := model.CountUserSubscriptionsByPlan(userId, plan.Id)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if count >= int64(plan.MaxPurchasePerUser) {
			common.ApiErrorMsg(c, "已达到该套餐购买上限")
			return
		}
	}

	client, _, clientErr := getWechatPayClient()
	if clientErr != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付订阅 client 初始化失败 error=%q", clientErr.Error()))
		common.ApiErrorMsg(c, "微信支付配置错误")
		return
	}

	callBackAddress := service.GetCallbackAddress()
	notifyUrl := callBackAddress + "/api/subscription/wechat/notify"
	tradeNo := fmt.Sprintf("%s%d", common.GetRandomString(6), time.Now().Unix())
	tradeNo = fmt.Sprintf("SUBWX%dNO%s", userId, tradeNo)

	dPayMoney := decimal.NewFromFloat(plan.PriceAmount).Mul(decimal.NewFromInt(100))
	amountInFen := dPayMoney.IntPart()

	description := fmt.Sprintf("SUB:%s", plan.Title)
	ctx := context.Background()

	order := &model.SubscriptionOrder{
		UserId:          userId,
		PlanId:          plan.Id,
		Money:           plan.PriceAmount,
		TradeNo:         tradeNo,
		PaymentMethod:   req.PaymentMethod,
		PaymentProvider: model.PaymentProviderWechatPay,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := order.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付订阅 创建订单失败 user_id=%d trade_no=%s error=%q", userId, tradeNo, err.Error()))
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}

	if req.PaymentMethod == model.PaymentMethodWechatNative {
		svc := &native.NativeApiService{Client: client}
		resp, _, err := svc.Prepay(ctx, native.PrepayRequest{
			Description: core.String(description),
			OutTradeNo:  core.String(tradeNo),
			NotifyUrl:   core.String(notifyUrl),
			Amount: &native.Amount{
				Total:    core.Int64(amountInFen),
				Currency: core.String("CNY"),
			},
		})
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付订阅 Native 下单失败 user_id=%d trade_no=%s error=%q", userId, tradeNo, err.Error()))
			_ = model.ExpireSubscriptionOrder(tradeNo, model.PaymentProviderWechatPay)
			common.ApiErrorMsg(c, "拉起支付失败")
			return
		}

		codeUrl := ""
		if resp.CodeUrl != nil {
			codeUrl = *resp.CodeUrl
		}

		logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付订阅 Native 订单创建成功 user_id=%d trade_no=%s plan_id=%d money=%.2f", userId, tradeNo, plan.Id, plan.PriceAmount))
		c.JSON(http.StatusOK, gin.H{"message": "success", "type": "native", "code_url": codeUrl})
	} else {
		returnUrl := system_setting.ServerAddress + "/console/topup"
		svc := &h5.H5ApiService{Client: client}
		payerClientIp := c.ClientIP()
		if payerClientIp == "" {
			payerClientIp = "127.0.0.1"
		}
		resp, _, err := svc.Prepay(ctx, h5.PrepayRequest{
			Description: core.String(description),
			OutTradeNo:  core.String(tradeNo),
			NotifyUrl:   core.String(notifyUrl),
			Amount: &h5.Amount{
				Total:    core.Int64(amountInFen),
				Currency: core.String("CNY"),
			},
			SceneInfo: &h5.SceneInfo{
				PayerClientIp: core.String(payerClientIp),
				H5Info: &h5.H5Info{
					Type:   core.String("Wap"),
					AppUrl: core.String(returnUrl),
				},
			},
		})
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付订阅 H5 下单失败 user_id=%d trade_no=%s error=%q", userId, tradeNo, err.Error()))
			_ = model.ExpireSubscriptionOrder(tradeNo, model.PaymentProviderWechatPay)
			common.ApiErrorMsg(c, "拉起支付失败")
			return
		}

		h5Url := ""
		if resp.H5Url != nil {
			h5Url = *resp.H5Url
		}

		logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付订阅 H5 订单创建成功 user_id=%d trade_no=%s plan_id=%d money=%.2f", userId, tradeNo, plan.Id, plan.PriceAmount))
		c.JSON(http.StatusOK, gin.H{"message": "success", "type": "h5", "h5_url": h5Url})
	}
}

func SubscriptionWechatNotify(c *gin.Context) {
	if !isWechatTopUpEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付订阅 webhook 被拒绝 reason=webhook_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "webhook disabled"})
		return
	}

	_, handler, err := getWechatPayClient()
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付订阅 client 初始化失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "config error"})
		return
	}

	transaction := new(payments.Transaction)
	notifyReq, err := handler.ParseNotifyRequest(context.Background(), c.Request, transaction)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付订阅 解析/验签通知失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "parse failed"})
		return
	}

	if notifyReq == nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付订阅 验签失败 notify_req=nil path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "verify failed"})
		return
	}

	tradeState := ""
	if transaction.TradeState != nil {
		tradeState = *transaction.TradeState
	}
	outTradeNo := ""
	if transaction.OutTradeNo != nil {
		outTradeNo = *transaction.OutTradeNo
	}
	transactionId := ""
	if transaction.TransactionId != nil {
		transactionId = *transaction.TransactionId
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付订阅 webhook 收到通知 transaction_id=%s out_trade_no=%s trade_state=%s client_ip=%s", transactionId, outTradeNo, tradeState, c.ClientIP()))

	if tradeState != "SUCCESS" {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付订阅 webhook 忽略事件 transaction_id=%s trade_state=%s client_ip=%s", transactionId, tradeState, c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})
		return
	}

	tradeNo := outTradeNo
	if tradeNo == "" {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付订阅 回调缺少out_trade_no client_ip=%s", c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "missing out_trade_no"})
		return
	}

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	order := model.GetSubscriptionOrderByTradeNo(tradeNo)
	if order == nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付订阅 回调订单不存在 trade_no=%s client_ip=%s", tradeNo, c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})
		return
	}

	if order.PaymentProvider != model.PaymentProviderWechatPay {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付订阅 订单支付网关不匹配 trade_no=%s order_provider=%s client_ip=%s", tradeNo, order.PaymentProvider, c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})
		return
	}

	if order.Status != common.TopUpStatusPending {
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})
		return
	}

	if transaction.Amount == nil || transaction.Amount.Total == nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付订阅 回调缺少金额 trade_no=%s client_ip=%s", tradeNo, c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "missing amount"})
		return
	}

	callbackAmountInFen := *transaction.Amount.Total
	dOrderMoney := decimal.NewFromFloat(order.Money)
	dExpectedFen := dOrderMoney.Mul(decimal.NewFromInt(100))
	dCallbackFen := decimal.NewFromInt(int64(callbackAmountInFen))
	if !dCallbackFen.Equal(dExpectedFen) {
		callbackMoney := dCallbackFen.Div(decimal.NewFromInt(100)).InexactFloat64()
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付订阅 金额不一致 trade_no=%s order_money=%.2f callback_money=%.2f client_ip=%s", tradeNo, order.Money, callbackMoney, c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "amount mismatch"})
		return
	}

	payload := map[string]string{
		"transaction_id": transactionId,
		"out_trade_no":   outTradeNo,
		"trade_state":    tradeState,
	}

	if err := model.CompleteSubscriptionOrder(tradeNo, common.GetJsonString(payload), model.PaymentProviderWechatPay, order.PaymentMethod); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付订阅 完成订单失败 trade_no=%s error=%q", tradeNo, err.Error()))
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "complete order failed"})
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付订阅 完成订单成功 trade_no=%s user_id=%d client_ip=%s", tradeNo, order.UserId, c.ClientIP()))
	c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})
}
