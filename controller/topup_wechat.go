package controller

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
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
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

var (
	wechatPayClient    *core.Client
	wechatNotifyHandler *notify.Handler
	wechatPayInitOnce  sync.Once
	wechatPayInitErr   error
	wechatPayMu        sync.RWMutex
)

func getWechatPayClient() (*core.Client, *notify.Handler, error) {
	wechatPayMu.RLock()
	if wechatPayClient != nil && wechatNotifyHandler != nil {
		client := wechatPayClient
		handler := wechatNotifyHandler
		wechatPayMu.RUnlock()
		return client, handler, nil
	}
	wechatPayMu.RUnlock()

	wechatPayMu.Lock()
	defer wechatPayMu.Unlock()

	if wechatPayClient != nil && wechatNotifyHandler != nil {
		return wechatPayClient, wechatNotifyHandler, nil
	}

	if setting.WechatPayMchID == "" || setting.WechatPayAPIv3Key == "" || setting.WechatPaySerialNo == "" || setting.WechatPayPrivateKey == "" {
		return nil, nil, fmt.Errorf("wechat pay config incomplete")
	}

	privateKey, err := utils.LoadPrivateKey(string([]byte(setting.WechatPayPrivateKey)))
	if err != nil {
		return nil, nil, fmt.Errorf("load wechat pay private key failed: %w", err)
	}

	opts := []core.ClientOption{
		option.WithMerchantCredential(setting.WechatPayMchID, setting.WechatPaySerialNo, privateKey),
		option.WithWechatPayAutoAuthCipher(setting.WechatPayMchID, setting.WechatPaySerialNo, privateKey, setting.WechatPayAPIv3Key),
	}

	client, err := core.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("create wechat pay client failed: %w", err)
	}

	mgr := downloader.MgrInstance()
	err = mgr.RegisterDownloaderWithPrivateKey(context.Background(), privateKey, setting.WechatPaySerialNo, setting.WechatPayMchID, setting.WechatPayAPIv3Key)
	if err != nil {
		return nil, nil, fmt.Errorf("register wechat pay certificate downloader failed: %w", err)
	}
	certVisitor := mgr.GetCertificateVisitor(setting.WechatPayMchID)
	verifier := verifiers.NewSHA256WithRSAVerifier(certVisitor)
	handler, err := notify.NewRSANotifyHandler(setting.WechatPayAPIv3Key, verifier)
	if err != nil {
		return nil, nil, fmt.Errorf("create wechat pay notify handler failed: %w", err)
	}

	wechatPayClient = client
	wechatNotifyHandler = handler

	return client, handler, nil
}

func resetWechatPayClient() {
	wechatPayMu.Lock()
	wechatPayClient = nil
	wechatNotifyHandler = nil
	wechatPayMu.Unlock()
}

func init() {
	setting.OnWechatPayConfigChange = resetWechatPayClient
}

func isWechatTopUpEnabled() bool {
	return setting.WechatPayEnabled &&
		strings.TrimSpace(setting.WechatPayMchID) != "" &&
		strings.TrimSpace(setting.WechatPayAPIv3Key) != "" &&
		strings.TrimSpace(setting.WechatPaySerialNo) != "" &&
		strings.TrimSpace(setting.WechatPayPrivateKey) != ""
}

type WechatPayRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
}

func RequestWechatPay(c *gin.Context) {
	var req WechatPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}

	if req.Amount < int64(setting.WechatPayMinTopUp) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", setting.WechatPayMinTopUp)})
		return
	}

	if req.PaymentMethod != model.PaymentMethodWechatNative && req.PaymentMethod != model.PaymentMethodWechatH5 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付方式不存在"})
		return
	}

	if !isWechatTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "微信支付未启用"})
		return
	}

	client, _, err := getWechatPayClient()
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 client 初始化失败 error=%q", err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "微信支付配置错误"})
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
	notifyUrl := callBackAddress + "/api/wechat/notify"
	tradeNo := fmt.Sprintf("%s%d", common.GetRandomString(6), time.Now().Unix())
	tradeNo = fmt.Sprintf("TOPWX%dNO%s", id, tradeNo)

	dPayMoney := decimal.NewFromFloat(payMoney).Mul(decimal.NewFromInt(100))
	amountInFen := dPayMoney.IntPart()

	description := fmt.Sprintf("TUC%d", req.Amount)
	ctx := context.Background()

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
		PaymentProvider: model.PaymentProviderWechatPay,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 创建充值订单失败 user_id=%d trade_no=%s error=%q", id, tradeNo, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
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
			logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 Native 下单失败 user_id=%d trade_no=%s error=%q", id, tradeNo, err.Error()))
			c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
			return
		}

		codeUrl := ""
		if resp.CodeUrl != nil {
			codeUrl = *resp.CodeUrl
		}

		logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付 Native 充值订单创建成功 user_id=%d trade_no=%s amount=%d money=%.2f", id, tradeNo, req.Amount, payMoney))
		c.JSON(http.StatusOK, gin.H{"message": "success", "type": "native", "code_url": codeUrl, "trade_no": tradeNo})
	} else {
		returnUrl := system_setting.ServerAddress + "/console/log"
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
			logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 H5 下单失败 user_id=%d trade_no=%s error=%q", id, tradeNo, err.Error()))
			c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
			return
		}

		h5Url := ""
		if resp.H5Url != nil {
			h5Url = *resp.H5Url
		}

		logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付 H5 充值订单创建成功 user_id=%d trade_no=%s amount=%d money=%.2f", id, tradeNo, req.Amount, payMoney))
		c.JSON(http.StatusOK, gin.H{"message": "success", "type": "h5", "h5_url": h5Url, "trade_no": tradeNo})
	}
}

func RequestWechatAmount(c *gin.Context) {
	var req AmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}

	if req.Amount < int64(setting.WechatPayMinTopUp) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", setting.WechatPayMinTopUp)})
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

func WechatNotify(c *gin.Context) {
	if !isWechatTopUpEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付 webhook 被拒绝 reason=webhook_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "webhook disabled"})
		return
	}

	_, handler, err := getWechatPayClient()
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 client 初始化失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "config error"})
		return
	}

	transaction := new(payments.Transaction)
	notifyReq, err := handler.ParseNotifyRequest(context.Background(), c.Request, transaction)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 解析/验签通知失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "parse failed"})
		return
	}

	if notifyReq == nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 验签失败 notify_req=nil path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
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

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付 webhook 收到通知 transaction_id=%s out_trade_no=%s trade_state=%s client_ip=%s", transactionId, outTradeNo, tradeState, c.ClientIP()))

	if tradeState != "SUCCESS" {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付 webhook 忽略事件 transaction_id=%s trade_state=%s client_ip=%s", transactionId, tradeState, c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})
		return
	}

	tradeNo := outTradeNo
	if tradeNo == "" {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 回调缺少out_trade_no client_ip=%s", c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "missing out_trade_no"})
		return
	}

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付 回调订单不存在 trade_no=%s client_ip=%s", tradeNo, c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})
		return
	}

	if topUp.PaymentProvider != model.PaymentProviderWechatPay {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付 订单支付网关不匹配 trade_no=%s order_provider=%s client_ip=%s", tradeNo, topUp.PaymentProvider, c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})
		return
	}

	if topUp.Status != common.TopUpStatusPending {
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})
		return
	}

	if transaction.Amount == nil || transaction.Amount.Total == nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 回调缺少金额 trade_no=%s client_ip=%s", tradeNo, c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "missing amount"})
		return
	}

	callbackAmountInFen := *transaction.Amount.Total
	dOrderMoney := decimal.NewFromFloat(topUp.Money)
	dExpectedFen := dOrderMoney.Mul(decimal.NewFromInt(100))
	dCallbackFen := decimal.NewFromInt(int64(callbackAmountInFen))
	if !dCallbackFen.Equal(dExpectedFen) {
		callbackMoney := dCallbackFen.Div(decimal.NewFromInt(100)).InexactFloat64()
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 金额不一致 trade_no=%s order_money=%.2f callback_money=%.2f client_ip=%s", tradeNo, topUp.Money, callbackMoney, c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "amount mismatch"})
		return
	}

	if err := model.RechargeWechat(tradeNo, c.ClientIP()); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 充值失败 trade_no=%s user_id=%d client_ip=%s error=%q", tradeNo, topUp.UserId, c.ClientIP(), err.Error()))
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "recharge failed"})
		return
	}

	quotaToAdd := int(decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
	if quotaToAdd > 0 && topUp.Id > 0 {
		gopool.Go(func() { service.TriggerRebateOnRecharge(topUp.UserId, quotaToAdd, topUp.Id) })
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付 充值成功 trade_no=%s user_id=%d client_ip=%s money=%.2f", tradeNo, topUp.UserId, c.ClientIP(), topUp.Money))
	c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})
}

var _ *rsa.PrivateKey
