package service

import (
	"context"
	"fmt"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
	"paygate-omni/internal/model"
)

func (s *PayService) createAlipayOrder(ctx context.Context, order *model.Order, channel *model.PayChannel) (*UnifiedOrderResult, error) {
	isProd := !channel.IsSandbox
	client, err := alipay.NewClient(channel.AppID, channel.PrivateKey, isProd)
	if err != nil {
		return nil, fmt.Errorf("failed to init alipay client: %w", err)
	}

	notifyUrl := order.NotifyURL
	if notifyUrl == "" {
		notifyUrl = "https://paygate.example.com/api/v1/pay/notify/alipay"
	}

	bm := make(gopay.BodyMap)
	bm.Set("subject", order.Subject).
		Set("out_trade_no", order.TradeNo). // 我们系统的内部单号
		Set("total_amount", fmt.Sprintf("%.2f", float64(order.Amount)/100.0)). // 支付宝按元计算 (微信按分)
		Set("notify_url", notifyUrl)

	// 发起电脑网站支付或扫码支付。这里为了对齐微信 Native 模式，也用扫码付 TradePrecreate，会返回二维码 qr_code
	aliRsp, err := client.TradePrecreate(ctx, bm)
	if err != nil {
		return nil, fmt.Errorf("alipay TradePrecreate api failed: %w", err)
	}

	if aliRsp.Response.Code != "10000" {
		return nil, fmt.Errorf("alipay api error: %s - %s", aliRsp.Response.Msg, aliRsp.Response.SubMsg)
	}

	return &UnifiedOrderResult{
		TradeNo: order.TradeNo,
		PayParams: map[string]string{
			"qr_code": aliRsp.Response.QrCode, // 商户将此 URL 转为二维码供用户扫码
		},
	}, nil
}
