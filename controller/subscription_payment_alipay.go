package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/smartwalle/alipay/v3"
)

type SubscriptionAlipayPayRequest struct {
	PlanId        int    `json:"plan_id"`
	PaymentMethod string `json:"payment_method"`
}

func SubscriptionRequestAlipayPay(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	var req SubscriptionAlipayPayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	if req.PaymentMethod != model.PaymentMethodAlipayPage && req.PaymentMethod != model.PaymentMethodAlipayWap {
		common.ApiErrorMsg(c, "支付方式不存在")
		return
	}

	if !isAlipayTopUpEnabled() {
		common.ApiErrorMsg(c, "支付宝支付未启用")
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

	callBackAddress := service.GetCallbackAddress()
	returnUrl, urlErr := url.Parse(system_setting.ServerAddress + "/console/topup")
	if urlErr != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝订阅 解析returnUrl失败 error=%q", urlErr.Error()))
		returnUrl = &url.URL{}
	}
	notifyUrl, urlErr := url.Parse(callBackAddress + "/api/subscription/alipay/notify")
	if urlErr != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝订阅 解析notifyUrl失败 error=%q", urlErr.Error()))
		notifyUrl = &url.URL{}
	}

	tradeNo := fmt.Sprintf("%s%d", common.GetRandomString(6), time.Now().Unix())
	tradeNo = fmt.Sprintf("SUBALI%dNO%s", userId, tradeNo)

	client := getAlipayClient()
	if client == nil {
		common.ApiErrorMsg(c, "当前管理员未配置支付宝支付信息")
		return
	}

	payMoneyStr := strconv.FormatFloat(plan.PriceAmount, 'f', 2, 64)
	subject := fmt.Sprintf("SUB:%s", plan.Title)

	var payUrlStr string

	if req.PaymentMethod == model.PaymentMethodAlipayPage {
		p := alipay.TradePagePay{}
		p.OutTradeNo = tradeNo
		p.TotalAmount = payMoneyStr
		p.Subject = subject
		p.ProductCode = "FAST_INSTANT_TRADE_PAY"
		p.NotifyURL = notifyUrl.String()
		p.ReturnURL = returnUrl.String()

		res, err := client.TradePagePay(p)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝订阅 Page Pay 下单失败 user_id=%d trade_no=%s error=%q", userId, tradeNo, err.Error()))
			_ = model.ExpireSubscriptionOrder(tradeNo, model.PaymentProviderAlipay)
			common.ApiErrorMsg(c, "拉起支付失败")
			return
		}
		payUrlStr = res.String()
	} else {
		p := alipay.TradeWapPay{}
		p.OutTradeNo = tradeNo
		p.TotalAmount = payMoneyStr
		p.Subject = subject
		p.ProductCode = "QUICK_WAP_WAY"
		p.NotifyURL = notifyUrl.String()
		p.ReturnURL = returnUrl.String()

		res, err := client.TradeWapPay(p)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝订阅 WAP Pay 下单失败 user_id=%d trade_no=%s error=%q", userId, tradeNo, err.Error()))
			_ = model.ExpireSubscriptionOrder(tradeNo, model.PaymentProviderAlipay)
			common.ApiErrorMsg(c, "拉起支付失败")
			return
		}
		payUrlStr = res.String()
	}

	order := &model.SubscriptionOrder{
		UserId:          userId,
		PlanId:          plan.Id,
		Money:           plan.PriceAmount,
		TradeNo:         tradeNo,
		PaymentMethod:   req.PaymentMethod,
		PaymentProvider: model.PaymentProviderAlipay,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := order.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝订阅 创建订单失败 user_id=%d trade_no=%s error=%q", userId, tradeNo, err.Error()))
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝订阅 订单创建成功 user_id=%d trade_no=%s plan_id=%d money=%.2f method=%s", userId, tradeNo, plan.Id, plan.PriceAmount, req.PaymentMethod))
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": payUrlStr, "url": payUrlStr})
}

func SubscriptionAlipayNotify(c *gin.Context) {
	if !isAlipayTopUpEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝订阅 webhook 被拒绝 reason=webhook_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	client := getAlipayClient()
	if client == nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝订阅 client 未初始化 path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	notification, err := client.GetTradeNotification(c.Request)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝订阅 解析通知失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝订阅 webhook 收到通知 trade_no=%s out_trade_no=%s trade_status=%s client_ip=%s", notification.TradeNo, notification.OutTradeNo, notification.TradeStatus, c.ClientIP()))

	if notification.TradeStatus != alipay.TradeStatusSuccess && notification.TradeStatus != alipay.TradeStatusFinished {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝订阅 webhook 忽略事件 trade_no=%s trade_status=%s client_ip=%s", notification.TradeNo, notification.TradeStatus, c.ClientIP()))
		alipay.AckNotification(c.Writer)
		return
	}

	tradeNo := notification.OutTradeNo
	if tradeNo == "" {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝订阅 回调缺少out_trade_no client_ip=%s", c.ClientIP()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	order := model.GetSubscriptionOrderByTradeNo(tradeNo)
	if order == nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝订阅 回调订单不存在 trade_no=%s client_ip=%s", tradeNo, c.ClientIP()))
		alipay.AckNotification(c.Writer)
		return
	}

	if order.PaymentProvider != model.PaymentProviderAlipay {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝订阅 订单支付网关不匹配 trade_no=%s order_provider=%s client_ip=%s", tradeNo, order.PaymentProvider, c.ClientIP()))
		alipay.AckNotification(c.Writer)
		return
	}

	if order.Status != common.TopUpStatusPending {
		alipay.AckNotification(c.Writer)
		return
	}

	callbackMoney, err := strconv.ParseFloat(notification.TotalAmount, 64)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝订阅 解析回调金额失败 trade_no=%s total_amount=%s client_ip=%s error=%q", tradeNo, notification.TotalAmount, c.ClientIP(), err.Error()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	dOrderMoney := decimal.NewFromFloat(order.Money)
	dCallbackMoney := decimal.NewFromFloat(callbackMoney)
	if !dCallbackMoney.Equal(dOrderMoney) {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝订阅 金额不一致 trade_no=%s order_money=%.2f callback_money=%.2f client_ip=%s", tradeNo, order.Money, callbackMoney, c.ClientIP()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	payload := map[string]string{
		"trade_no":     notification.TradeNo,
		"out_trade_no": notification.OutTradeNo,
		"trade_status": string(notification.TradeStatus),
		"total_amount": notification.TotalAmount,
	}

	if err := model.CompleteSubscriptionOrder(tradeNo, common.GetJsonString(payload), model.PaymentProviderAlipay, order.PaymentMethod); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝订阅 完成订单失败 trade_no=%s error=%q", tradeNo, err.Error()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝订阅 完成订单成功 trade_no=%s user_id=%d client_ip=%s", tradeNo, order.UserId, c.ClientIP()))
	alipay.AckNotification(c.Writer)
}
