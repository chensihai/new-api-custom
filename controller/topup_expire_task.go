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
	topUpExpireBatchSize      = 100
	topUpExpireScanPeriod     = 1 * time.Minute
	topUpExpireStartDelay     = 10 * time.Second
	topUpExpireQueryTimeout   = 10 * time.Second
)

func StartTopUpExpireTask() {
	time.Sleep(topUpExpireStartDelay)
	ticker := time.NewTicker(topUpExpireScanPeriod)
	defer ticker.Stop()

	common.SysLog("充值订单超时关闭任务已启动")
	for {
		<-ticker.C
		runTopUpExpireOnce()
	}
}

func runTopUpExpireOnce() {
	now := common.GetTimestamp()
	maxCreateTime := now - common.TopUpOrderTimeoutSeconds

	var totalExpired int
	var consecutiveZeroSuccess int

	for {
		topUps, err := model.GetPendingTopUpsBefore(maxCreateTime, topUpExpireBatchSize)
		if err != nil {
			common.SysLog(fmt.Sprintf("充值订单超时关闭 查询失败 error=%q", err.Error()))
			return
		}
		if len(topUps) == 0 {
			break
		}

		var batchExpired int
		for i := range topUps {
			settled, shouldClose, err := queryAndSettleBeforeExpire(&topUps[i])
			if err != nil {
				common.SysLog(fmt.Sprintf("充值订单超时关闭 查询/补单异常 trade_no=%s error=%q", topUps[i].TradeNo, err.Error()))
			}
			if settled {
				batchExpired++
				continue
			}
			if !shouldClose {
				continue
			}
			if err := model.ExpireTopUpOrder(topUps[i].TradeNo); err != nil {
				common.SysLog(fmt.Sprintf("充值订单超时关闭 关闭失败 trade_no=%s error=%q", topUps[i].TradeNo, err.Error()))
				continue
			}
			batchExpired++
		}

		totalExpired += batchExpired
		if batchExpired == 0 {
			consecutiveZeroSuccess++
			if consecutiveZeroSuccess >= 2 {
				break
			}
		} else {
			consecutiveZeroSuccess = 0
		}

		if len(topUps) < topUpExpireBatchSize {
			break
		}
	}

	if totalExpired > 0 {
		common.SysLog(fmt.Sprintf("充值订单超时关闭 本轮关闭%d笔订单", totalExpired))
	}
}

func queryAndSettleBeforeExpire(topUp *model.TopUp) (settled bool, shouldClose bool, err error) {
	if topUp.PaymentProvider != model.PaymentProviderAlipay && topUp.PaymentProvider != model.PaymentProviderWechatPay {
		return false, true, nil
	}

	LockOrder(topUp.TradeNo)
	defer UnlockOrder(topUp.TradeNo)

	freshTopUp := model.GetTopUpByTradeNo(topUp.TradeNo)
	if freshTopUp == nil || freshTopUp.Status != common.TopUpStatusPending {
		return false, false, nil
	}

	switch topUp.PaymentProvider {
	case model.PaymentProviderAlipay:
		return queryAndSettleAlipay(freshTopUp)
	case model.PaymentProviderWechatPay:
		return queryAndSettleWechat(freshTopUp)
	default:
		return false, true, nil
	}
}

func queryAndSettleAlipay(topUp *model.TopUp) (bool, bool, error) {
	if !IsAlipayPollingEnabled() {
		return false, true, nil
	}

	client := GetAlipayClient()
	if client == nil {
		return false, false, fmt.Errorf("alipay client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), topUpExpireQueryTimeout)
	defer cancel()

	rsp, err := client.TradeQuery(ctx, alipay.TradeQuery{OutTradeNo: topUp.TradeNo})
	if err != nil {
		return false, false, fmt.Errorf("alipay trade query failed: %w", err)
	}

	if rsp.TradeStatus != alipay.TradeStatusSuccess && rsp.TradeStatus != alipay.TradeStatusFinished {
		return false, true, nil
	}

	callbackMoney, parseErr := strconv.ParseFloat(rsp.TotalAmount, 64)
	if parseErr != nil {
		common.SysLog(fmt.Sprintf("充值订单超时关闭 支付宝解析金额失败 trade_no=%s total_amount=%s error=%q", topUp.TradeNo, rsp.TotalAmount, parseErr.Error()))
		return false, true, nil
	}

	dOrderMoney := decimal.NewFromFloat(topUp.Money)
	dCallbackMoney := decimal.NewFromFloat(callbackMoney)
	if !dCallbackMoney.Equal(dOrderMoney) {
		common.SysLog(fmt.Sprintf("充值订单超时关闭 支付宝金额不一致 trade_no=%s order_money=%.2f callback_money=%.2f", topUp.TradeNo, topUp.Money, callbackMoney))
		return false, true, nil
	}

	if err := model.RechargeAlipay(topUp.TradeNo, "expire-task"); err != nil {
		common.SysLog(fmt.Sprintf("充值订单超时关闭 支付宝补单事务失败 trade_no=%s error=%q", topUp.TradeNo, err.Error()))
		return false, false, err
	}

	quotaToAdd := int(decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
	if quotaToAdd > 0 && topUp.Id > 0 {
		userId := topUp.UserId
		topUpId := topUp.Id
		gopool.Go(func() { model.TriggerRebateOnRechargeAsync(userId, quotaToAdd, topUpId) })
	}

	common.SysLog(fmt.Sprintf("充值订单超时关闭 支付宝补单成功 trade_no=%s user_id=%d money=%.2f", topUp.TradeNo, topUp.UserId, topUp.Money))
	return true, false, nil
}

func queryAndSettleWechat(topUp *model.TopUp) (bool, bool, error) {
	if !IsWechatTopUpEnabled() {
		return false, true, nil
	}

	client, _, err := GetWechatPayClient()
	if err != nil {
		return false, false, fmt.Errorf("wechat pay client not initialized: %w", err)
	}

	svc := &native.NativeApiService{Client: client}

	ctx, cancel := context.WithTimeout(context.Background(), topUpExpireQueryTimeout)
	defer cancel()

	resp, _, err := svc.QueryOrderByOutTradeNo(ctx, native.QueryOrderByOutTradeNoRequest{
		OutTradeNo: core.String(topUp.TradeNo),
		Mchid:      core.String(setting.WechatPayMchID),
	})
	if err != nil {
		return false, false, fmt.Errorf("wechat pay query failed: %w", err)
	}

	tradeState := ""
	if resp.TradeState != nil {
		tradeState = *resp.TradeState
	}
	if tradeState != "SUCCESS" {
		return false, true, nil
	}

	if resp.Amount == nil || resp.Amount.Total == nil {
		common.SysLog(fmt.Sprintf("充值订单超时关闭 微信支付缺少金额 trade_no=%s", topUp.TradeNo))
		return false, true, nil
	}

	callbackAmountInFen := *resp.Amount.Total
	dOrderMoney := decimal.NewFromFloat(topUp.Money)
	dExpectedFen := dOrderMoney.Mul(decimal.NewFromInt(100))
	dCallbackFen := decimal.NewFromInt(int64(callbackAmountInFen))
	if !dCallbackFen.Equal(dExpectedFen) {
		callbackMoney := dCallbackFen.Div(decimal.NewFromInt(100)).InexactFloat64()
		common.SysLog(fmt.Sprintf("充值订单超时关闭 微信支付金额不一致 trade_no=%s order_money=%.2f callback_money=%.2f", topUp.TradeNo, topUp.Money, callbackMoney))
		return false, true, nil
	}

	if err := model.RechargeWechat(topUp.TradeNo, "expire-task"); err != nil {
		common.SysLog(fmt.Sprintf("充值订单超时关闭 微信支付补单事务失败 trade_no=%s error=%q", topUp.TradeNo, err.Error()))
		return false, false, err
	}

	quotaToAdd := int(decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
	if quotaToAdd > 0 && topUp.Id > 0 {
		userId := topUp.UserId
		topUpId := topUp.Id
		gopool.Go(func() { model.TriggerRebateOnRechargeAsync(userId, quotaToAdd, topUpId) })
	}

	common.SysLog(fmt.Sprintf("充值订单超时关闭 微信支付补单成功 trade_no=%s user_id=%d money=%.2f", topUp.TradeNo, topUp.UserId, topUp.Money))
	return true, false, nil
}
