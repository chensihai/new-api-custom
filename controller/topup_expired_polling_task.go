package controller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/shopspring/decimal"
	"github.com/smartwalle/alipay/v3"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
)

const (
	topUpExpiredPollingWindowHours  = 24
	topUpExpiredPollingPeriod       = 5 * time.Minute
	topUpExpiredPollingBatchSize    = 50
	topUpExpiredPollingStartDelay   = 15 * time.Second
	topUpExpiredPollingQueryTimeout = 10 * time.Second
)

func StartTopUpExpiredPollingTask() {
	time.Sleep(topUpExpiredPollingStartDelay)
	ticker := time.NewTicker(topUpExpiredPollingPeriod)
	defer ticker.Stop()

	common.SysLog("expired订单掉单查询任务已启动")
	for {
		<-ticker.C
		runExpiredPollingOnce()
	}
}

func runExpiredPollingOnce() {
	now := common.GetTimestamp()
	windowStart := now - int64(topUpExpiredPollingWindowHours)*3600

	for {
		topUps, err := model.GetExpiredTopUpsInWindow(windowStart, topUpExpiredPollingBatchSize)
		if err != nil {
			common.SysLog(fmt.Sprintf("expired订单掉单查询 获取订单失败 error=%q", err.Error()))
			return
		}
		if len(topUps) == 0 {
			return
		}

		for i := range topUps {
			pollExpiredOrder(&topUps[i])
		}

		if len(topUps) < topUpExpiredPollingBatchSize {
			return
		}
	}
}

func pollExpiredOrder(topUp *model.TopUp) {
	if topUp.PaymentProvider != model.PaymentProviderAlipay && topUp.PaymentProvider != model.PaymentProviderWechatPay {
		return
	}

	LockOrder(topUp.TradeNo)
	defer UnlockOrder(topUp.TradeNo)

	freshTopUp := model.GetTopUpByTradeNo(topUp.TradeNo)
	if freshTopUp == nil {
		return
	}
	if freshTopUp.Status == common.TopUpStatusSuccess {
		return
	}
	if freshTopUp.Status != common.TopUpStatusExpired {
		return
	}

	switch topUp.PaymentProvider {
	case model.PaymentProviderAlipay:
		pollExpiredAlipayOrder(freshTopUp)
	case model.PaymentProviderWechatPay:
		pollExpiredWechatOrder(freshTopUp)
	default:
	}
}

func pollExpiredAlipayOrder(topUp *model.TopUp) {
	if !IsAlipayPollingEnabled() {
		return
	}

	client := GetAlipayClient()
	if client == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), topUpExpiredPollingQueryTimeout)
	defer cancel()

	rsp, err := client.TradeQuery(ctx, alipay.TradeQuery{OutTradeNo: topUp.TradeNo})
	if err != nil {
		common.SysLog(fmt.Sprintf("expired订单掉单查询 支付宝查询失败 trade_no=%s error=%q", topUp.TradeNo, err.Error()))
		return
	}

	if rsp.TradeStatus != alipay.TradeStatusSuccess && rsp.TradeStatus != alipay.TradeStatusFinished {
		return
	}

	callbackMoney, parseErr := strconv.ParseFloat(rsp.TotalAmount, 64)
	if parseErr != nil {
		common.SysLog(fmt.Sprintf("expired订单掉单查询 支付宝解析金额失败 trade_no=%s total_amount=%s error=%q", topUp.TradeNo, rsp.TotalAmount, parseErr.Error()))
		return
	}

	dOrderMoney := decimal.NewFromFloat(topUp.Money)
	dCallbackMoney := decimal.NewFromFloat(callbackMoney)
	if !dCallbackMoney.Equal(dOrderMoney) {
		common.SysLog(fmt.Sprintf("expired订单掉单查询 支付宝金额不一致 trade_no=%s order_money=%.2f callback_money=%.2f", topUp.TradeNo, topUp.Money, callbackMoney))
		return
	}

	if err := model.RechargeAlipay(topUp.TradeNo, "expired-polling"); err != nil {
		common.SysLog(fmt.Sprintf("expired订单掉单查询 支付宝补单事务失败 trade_no=%s error=%q", topUp.TradeNo, err.Error()))
		return
	}

	quotaToAdd := int(decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
	if quotaToAdd > 0 && topUp.Id > 0 {
		userId := topUp.UserId
		topUpId := topUp.Id
		gopool.Go(func() { model.TriggerRebateOnRechargeAsync(userId, quotaToAdd, topUpId) })
	}

	common.SysLog(fmt.Sprintf("expired订单掉单查询 支付宝补单成功 trade_no=%s user_id=%d money=%.2f", topUp.TradeNo, topUp.UserId, topUp.Money))
}

func pollExpiredWechatOrder(topUp *model.TopUp) {
	if !IsWechatTopUpEnabled() {
		return
	}

	client, _, err := GetWechatPayClient()
	if err != nil {
		return
	}

	svc := &native.NativeApiService{Client: client}

	ctx, cancel := context.WithTimeout(context.Background(), topUpExpiredPollingQueryTimeout)
	defer cancel()

	resp, _, err := svc.QueryOrderByOutTradeNo(ctx, native.QueryOrderByOutTradeNoRequest{
		OutTradeNo: core.String(topUp.TradeNo),
		Mchid:      core.String(setting.WechatPayMchID),
	})
	if err != nil {
		common.SysLog(fmt.Sprintf("expired订单掉单查询 微信支付查询失败 trade_no=%s error=%q", topUp.TradeNo, err.Error()))
		return
	}

	tradeState := ""
	if resp.TradeState != nil {
		tradeState = *resp.TradeState
	}
	if tradeState != "SUCCESS" {
		return
	}

	if resp.Amount == nil || resp.Amount.Total == nil {
		common.SysLog(fmt.Sprintf("expired订单掉单查询 微信支付缺少金额 trade_no=%s", topUp.TradeNo))
		return
	}

	callbackAmountInFen := *resp.Amount.Total
	dOrderMoney := decimal.NewFromFloat(topUp.Money)
	dExpectedFen := dOrderMoney.Mul(decimal.NewFromInt(100))
	dCallbackFen := decimal.NewFromInt(int64(callbackAmountInFen))
	if !dCallbackFen.Equal(dExpectedFen) {
		callbackMoney := dCallbackFen.Div(decimal.NewFromInt(100)).InexactFloat64()
		common.SysLog(fmt.Sprintf("expired订单掉单查询 微信支付金额不一致 trade_no=%s order_money=%.2f callback_money=%.2f", topUp.TradeNo, topUp.Money, callbackMoney))
		return
	}

	if err := model.RechargeWechat(topUp.TradeNo, "expired-polling"); err != nil {
		common.SysLog(fmt.Sprintf("expired订单掉单查询 微信支付补单事务失败 trade_no=%s error=%q", topUp.TradeNo, err.Error()))
		return
	}

	quotaToAdd := int(decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
	if quotaToAdd > 0 && topUp.Id > 0 {
		userId := topUp.UserId
		topUpId := topUp.Id
		gopool.Go(func() { model.TriggerRebateOnRechargeAsync(userId, quotaToAdd, topUpId) })
	}

	common.SysLog(fmt.Sprintf("expired订单掉单查询 微信支付补单成功 trade_no=%s user_id=%d money=%.2f", topUp.TradeNo, topUp.UserId, topUp.Money))
}
