package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/smartwalle/alipay/v3"
)

var (
	alipayClient *alipay.Client
	alipayMu     sync.RWMutex
)

func GetAlipayClient() *alipay.Client {
	alipayMu.RLock()
	if alipayClient != nil {
		client := alipayClient
		alipayMu.RUnlock()
		return client
	}
	alipayMu.RUnlock()

	alipayMu.Lock()
	defer alipayMu.Unlock()

	if alipayClient != nil {
		return alipayClient
	}

	if setting.AlipayAppId == "" || setting.AlipayPrivateKey == "" || setting.AlipayPublicKey == "" {
		return nil
	}
	client, err := alipay.New(setting.AlipayAppId, setting.AlipayPrivateKey, !setting.AlipaySandbox)
	if err != nil {
		common.SysError("failed to create alipay client: " + err.Error())
		return nil
	}
	if err := client.LoadAliPayPublicKey(setting.AlipayPublicKey); err != nil {
		common.SysError("failed to load alipay public key: " + err.Error())
		return nil
	}
	alipayClient = client
	return alipayClient
}

func resetAlipayClient() {
	alipayMu.Lock()
	alipayClient = nil
	alipayMu.Unlock()
}

func init() {
	setting.OnAlipayConfigChange = resetAlipayClient
}

func IsAlipayTopUpEnabled() bool {
	return setting.AlipayEnabled &&
		strings.TrimSpace(setting.AlipayAppId) != "" &&
		strings.TrimSpace(setting.AlipayPrivateKey) != "" &&
		strings.TrimSpace(setting.AlipayPublicKey) != ""
}

type AlipayPayRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
}

func RequestAlipayPay(c *gin.Context) {
	var req AlipayPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}

	if req.Amount < int64(setting.AlipayMinTopUp) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", setting.AlipayMinTopUp)})
		return
	}

	if req.PaymentMethod != model.PaymentMethodAlipayPage && req.PaymentMethod != model.PaymentMethodAlipayWap {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付方式不存在"})
		return
	}

	if !IsAlipayTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付宝支付未启用"})
		return
	}

	id := c.GetInt("id")

	pendingCount, err := model.CountPendingTopUpsByUserId(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "系统繁忙，请稍后重试"})
		return
	}
	if pendingCount >= common.MaxPendingTopUpOrders {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "未支付订单数量已达上限"})
		return
	}
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	if payMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	callBackAddress := service.GetCallbackAddress()
	returnUrl, urlErr := url.Parse(system_setting.ServerAddress + "/console/log")
	if urlErr != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝 解析returnUrl失败 error=%q", urlErr.Error()))
		returnUrl = &url.URL{}
	}
	notifyUrl, urlErr := url.Parse(callBackAddress + "/api/alipay/notify")
	if urlErr != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝 解析notifyUrl失败 error=%q", urlErr.Error()))
		notifyUrl = &url.URL{}
	}
	tradeNo := fmt.Sprintf("%s%d", common.GetRandomString(6), time.Now().Unix())
	tradeNo = fmt.Sprintf("TOPALI%dNO%s", id, tradeNo)

	client := GetAlipayClient()
	if client == nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "当前管理员未配置支付宝支付信息"})
		return
	}

	var payUrlStr string
	payMoneyStr := strconv.FormatFloat(payMoney, 'f', 2, 64)

	if req.PaymentMethod == model.PaymentMethodAlipayPage {
		p := alipay.TradePagePay{}
		p.OutTradeNo = tradeNo
		p.TotalAmount = payMoneyStr
		p.Subject = fmt.Sprintf("TUC%d", req.Amount)
		p.ProductCode = "FAST_INSTANT_TRADE_PAY"
		p.NotifyURL = notifyUrl.String()
		p.ReturnURL = returnUrl.String()

		res, err := client.TradePagePay(p)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝 Page Pay 下单失败 user_id=%d trade_no=%s error=%q", id, tradeNo, err.Error()))
			c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
			return
		}
		payUrlStr = res.String()
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝 Page Pay URL user_id=%d pay_url=%s", id, payUrlStr))
	} else {
		p := alipay.TradeWapPay{}
		p.OutTradeNo = tradeNo
		p.TotalAmount = payMoneyStr
		p.Subject = fmt.Sprintf("TUC%d", req.Amount)
		p.ProductCode = "QUICK_WAP_WAY"
		p.NotifyURL = notifyUrl.String()
		p.ReturnURL = returnUrl.String()

		res, err := client.TradeWapPay(p)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝 WAP Pay 下单失败 user_id=%d trade_no=%s error=%q", id, tradeNo, err.Error()))
			c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
			return
		}
		payUrlStr = res.String()
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝 WAP Pay URL user_id=%d pay_url=%s", id, payUrlStr))
	}

	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount := decimal.NewFromInt(int64(amount))
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		amount = dAmount.Div(dQuotaPerUnit).IntPart()
	}

	topUp := &model.TopUp{
		UserId:          id,
		Amount:          amount,
		Money:           payMoney,
		TradeNo:         tradeNo,
		PaymentMethod:   req.PaymentMethod,
		PaymentProvider: model.PaymentProviderAlipay,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝 创建充值订单失败 user_id=%d trade_no=%s error=%q", id, tradeNo, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝 充值订单创建成功 user_id=%d trade_no=%s payment_method=%s amount=%d money=%.2f", id, tradeNo, req.PaymentMethod, req.Amount, payMoney))
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": payUrlStr, "url": payUrlStr, "trade_no": tradeNo})
}

func RequestAlipayAmount(c *gin.Context) {
	var req AmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}

	if req.Amount < int64(setting.AlipayMinTopUp) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", setting.AlipayMinTopUp)})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	if payMoney <= 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": strconv.FormatFloat(payMoney, 'f', 2, 64)})
}

func AlipayNotify(c *gin.Context) {
	if !IsAlipayTopUpEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝 webhook 被拒绝 reason=webhook_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	client := GetAlipayClient()
	if client == nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝 client 未初始化 path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	notification, err := client.GetTradeNotification(c.Request)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝 解析通知失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝 webhook 收到通知 trade_no=%s out_trade_no=%s trade_status=%s client_ip=%s", notification.TradeNo, notification.OutTradeNo, notification.TradeStatus, c.ClientIP()))

	if notification.TradeStatus != alipay.TradeStatusSuccess && notification.TradeStatus != alipay.TradeStatusFinished {
		if notification.TradeStatus == alipay.TradeStatusClosed {
			tradeNo := notification.OutTradeNo
			if tradeNo != "" {
				LockOrder(tradeNo)
				if err := model.MarkTopUpFailed(tradeNo, model.PaymentProviderAlipay, "支付宝交易关闭"); err != nil {
					logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝 webhook 标记失败失败 trade_no=%s error=%q", tradeNo, err.Error()))
				} else {
					logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝 webhook 交易关闭 trade_no=%s client_ip=%s", tradeNo, c.ClientIP()))
				}
				UnlockOrder(tradeNo)
			}
		} else {
			logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝 webhook 忽略事件 trade_no=%s trade_status=%s client_ip=%s", notification.TradeNo, notification.TradeStatus, c.ClientIP()))
		}
		alipay.AckNotification(c.Writer)
		return
	}

	tradeNo := notification.OutTradeNo
	if tradeNo == "" {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝 回调缺少out_trade_no client_ip=%s", c.ClientIP()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝 回调订单不存在 trade_no=%s client_ip=%s", tradeNo, c.ClientIP()))
		alipay.AckNotification(c.Writer)
		return
	}

	if topUp.PaymentProvider != model.PaymentProviderAlipay {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝 订单支付网关不匹配 trade_no=%s order_provider=%s client_ip=%s", tradeNo, topUp.PaymentProvider, c.ClientIP()))
		alipay.AckNotification(c.Writer)
		return
	}

	if topUp.Status != common.TopUpStatusPending {
		alipay.AckNotification(c.Writer)
		return
	}

	callbackMoney, err := strconv.ParseFloat(notification.TotalAmount, 64)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝 解析回调金额失败 trade_no=%s total_amount=%s client_ip=%s error=%q", tradeNo, notification.TotalAmount, c.ClientIP(), err.Error()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	dOrderMoney := decimal.NewFromFloat(topUp.Money)
	dCallbackMoney := decimal.NewFromFloat(callbackMoney)
	if !dCallbackMoney.Equal(dOrderMoney) {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝 金额不一致 trade_no=%s order_money=%.2f callback_money=%.2f client_ip=%s", tradeNo, topUp.Money, callbackMoney, c.ClientIP()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	if err := model.RechargeAlipay(tradeNo, c.ClientIP()); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝 充值失败 trade_no=%s user_id=%d client_ip=%s error=%q", tradeNo, topUp.UserId, c.ClientIP(), err.Error()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	quotaToAdd := int(decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
	if quotaToAdd > 0 && topUp.Id > 0 {
		gopool.Go(func() { service.TriggerRebateOnRecharge(topUp.UserId, quotaToAdd, topUp.Id) })
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝 充值成功 trade_no=%s user_id=%d client_ip=%s money=%.2f", tradeNo, topUp.UserId, c.ClientIP(), topUp.Money))
	alipay.AckNotification(c.Writer)
}
