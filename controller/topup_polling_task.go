package controller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/shopspring/decimal"
	"github.com/smartwalle/alipay/v3"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
)

const (
	topUpPollingBatchSize    = 50
	topUpPollingMinAge       = 60
	topUpPollingMaxAge       = 840
	topUpPollingSettlePeriod = 3 * time.Minute
)

func StartTopUpPollingTask() {
	time.Sleep(10 * time.Second)
	ticker := time.NewTicker(topUpPollingSettlePeriod)
	defer ticker.Stop()

	common.SysLog("支付掉单主动查询任务已启动")
	for {
		<-ticker.C
		runTopUpPollingOnce()
	}
}

func runTopUpPollingOnce() {
	now := common.GetTimestamp()
	maxCreateTime := now - topUpPollingMinAge
	minCreateTime := now - topUpPollingMaxAge

	for {
		topUps, err := model.GetPendingTopUpsBefore(maxCreateTime, topUpPollingBatchSize)
		if err != nil {
			common.SysLog(fmt.Sprintf("支付掉单查询 获取待查订单失败 error=%q", err.Error()))
			return
		}
		if len(topUps) == 0 {
			return
		}

		for i := range topUps {
			if topUps[i].CreateTime < minCreateTime {
				continue
			}
			pollTopUpOrder(&topUps[i])
		}

		if len(topUps) < topUpPollingBatchSize {
			return
		}
		maxCreateTime = now - topUpPollingMinAge
	}
}

func pollTopUpOrder(topUp *model.TopUp) {
	switch topUp.PaymentProvider {
	case model.PaymentProviderAlipay:
		pollAlipayOrder(topUp)
	case model.PaymentProviderWechatPay:
		pollWechatOrder(topUp)
	default:
	}
}

func pollAlipayOrder(topUp *model.TopUp) {
	if !IsAlipayPollingEnabled() {
		return
	}

	client := GetAlipayClient()
	if client == nil {
		return
	}

	LockOrder(topUp.TradeNo)
	defer UnlockOrder(topUp.TradeNo)

	freshTopUp := model.GetTopUpByTradeNo(topUp.TradeNo)
	if freshTopUp == nil || freshTopUp.Status != common.TopUpStatusPending {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rsp, err := client.TradeQuery(ctx, alipay.TradeQuery{OutTradeNo: topUp.TradeNo})
	if err != nil {
		common.SysLog(fmt.Sprintf("支付掉单查询 支付宝查询失败 trade_no=%s error=%q", topUp.TradeNo, err.Error()))
		return
	}

	if rsp.TradeStatus != alipay.TradeStatusSuccess && rsp.TradeStatus != alipay.TradeStatusFinished {
		return
	}

	callbackMoney, err := strconv.ParseFloat(rsp.TotalAmount, 64)
	if err != nil {
		common.SysLog(fmt.Sprintf("支付掉单查询 支付宝解析金额失败 trade_no=%s total_amount=%s error=%q", topUp.TradeNo, rsp.TotalAmount, err.Error()))
		return
	}

	dOrderMoney := decimal.NewFromFloat(freshTopUp.Money)
	dCallbackMoney := decimal.NewFromFloat(callbackMoney)
	if !dCallbackMoney.Equal(dOrderMoney) {
		common.SysLog(fmt.Sprintf("支付掉单查询 支付宝金额不一致 trade_no=%s order_money=%.2f callback_money=%.2f", topUp.TradeNo, freshTopUp.Money, callbackMoney))
		return
	}

	if err := model.RechargeAlipay(topUp.TradeNo, "polling"); err != nil {
		common.SysLog(fmt.Sprintf("支付掉单查询 支付宝充值失败 trade_no=%s user_id=%d error=%q", topUp.TradeNo, freshTopUp.UserId, err.Error()))
		return
	}

	quotaToAdd := int(decimal.NewFromInt(freshTopUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
	if quotaToAdd > 0 && freshTopUp.Id > 0 {
		gopool.Go(func() { service.TriggerRebateOnRecharge(freshTopUp.UserId, quotaToAdd, freshTopUp.Id) })
	}

	common.SysLog(fmt.Sprintf("支付掉单查询 支付宝补单成功 trade_no=%s user_id=%d money=%.2f", topUp.TradeNo, freshTopUp.UserId, freshTopUp.Money))
}

func IsAlipayPollingEnabled() bool {
	return setting.AlipayEnabled &&
		setting.AlipayAppId != "" &&
		setting.AlipayPrivateKey != "" &&
		setting.AlipayPublicKey != ""
}

func isWechatPollingEnabled() bool {
	return IsWechatTopUpEnabled()
}

func pollWechatOrder(topUp *model.TopUp) {
	if !isWechatPollingEnabled() {
		return
	}

	client, _, err := GetWechatPayClient()
	if err != nil {
		return
	}

	LockOrder(topUp.TradeNo)
	defer UnlockOrder(topUp.TradeNo)

	freshTopUp := model.GetTopUpByTradeNo(topUp.TradeNo)
	if freshTopUp == nil || freshTopUp.Status != common.TopUpStatusPending {
		return
	}

	svc := &native.NativeApiService{Client: client}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, _, err := svc.QueryOrderByOutTradeNo(ctx, native.QueryOrderByOutTradeNoRequest{
		OutTradeNo: core.String(topUp.TradeNo),
		Mchid:      core.String(setting.WechatPayMchID),
	})
	if err != nil {
		common.SysLog(fmt.Sprintf("支付掉单查询 微信支付查询失败 trade_no=%s error=%q", topUp.TradeNo, err.Error()))
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
		common.SysLog(fmt.Sprintf("支付掉单查询 微信支付缺少金额 trade_no=%s", topUp.TradeNo))
		return
	}

	callbackAmountInFen := *resp.Amount.Total
	dOrderMoney := decimal.NewFromFloat(freshTopUp.Money)
	dExpectedFen := dOrderMoney.Mul(decimal.NewFromInt(100))
	dCallbackFen := decimal.NewFromInt(int64(callbackAmountInFen))
	if !dCallbackFen.Equal(dExpectedFen) {
		callbackMoney := dCallbackFen.Div(decimal.NewFromInt(100)).InexactFloat64()
		common.SysLog(fmt.Sprintf("支付掉单查询 微信支付金额不一致 trade_no=%s order_money=%.2f callback_money=%.2f", topUp.TradeNo, freshTopUp.Money, callbackMoney))
		return
	}

	if err := model.RechargeWechat(topUp.TradeNo, "polling"); err != nil {
		common.SysLog(fmt.Sprintf("支付掉单查询 微信支付充值失败 trade_no=%s user_id=%d error=%q", topUp.TradeNo, freshTopUp.UserId, err.Error()))
		return
	}

	quotaToAdd := int(decimal.NewFromInt(freshTopUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
	if quotaToAdd > 0 && freshTopUp.Id > 0 {
		gopool.Go(func() { service.TriggerRebateOnRecharge(freshTopUp.UserId, quotaToAdd, freshTopUp.Id) })
	}

	common.SysLog(fmt.Sprintf("支付掉单查询 微信支付补单成功 trade_no=%s user_id=%d money=%.2f", topUp.TradeNo, freshTopUp.UserId, freshTopUp.Money))
}
