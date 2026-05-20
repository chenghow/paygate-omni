package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/wechat/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"paygate-omni/internal/model"
	"paygate-omni/internal/repository"
)

// PayService 封装支付核心业务。
type PayService struct {
	store  *repository.Store
	logger *zap.Logger
}

// NewPayService 创建 PayService 实例。
func NewPayService(store *repository.Store, logger *zap.Logger) *PayService {
	return &PayService{store: store, logger: logger}
}

// UnifiedOrderResult 表示统一下单结果。
type UnifiedOrderResult struct {
	TradeNo   string
	PayParams interface{}
}

// CreateOrder 处理统一下单。
func (s *PayService) CreateOrder(ctx context.Context, merchantAppID, merchantTradeNo, subject, channelType, notifyURL string, amount int64) (*UnifiedOrderResult, error) {
	// 1. 检查订单号是否已存在
	var count int64
	if err := s.store.DB.Model(&model.Order{}).Where("merchant_app_id = ? AND out_trade_no = ?", merchantAppID, merchantTradeNo).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("check duplicate order failed: %w", err)
	}
	if count > 0 {
		return nil, errors.New("merchant_trade_no already exists")
	}

	// 2. 获取支付渠道配置并根据主键自动解密出秘钥
	var payChannel model.PayChannel
	err := s.store.DB.Where("merchant_app_id = ? AND channel_type = ? AND is_active = true", merchantAppID, channelType).First(&payChannel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("active payment channel %s not found for merchant", channelType)
		}
		return nil, fmt.Errorf("query channel failed: %w", err)
	}

	// 3. 生成内部唯一流水号
	tradeNo := fmt.Sprintf("TR%d%d", time.Now().UnixNano(), time.Now().UnixMilli()%1000)

	// 4. 创建本地订单 (PENDING)
	order := model.Order{
		MerchantAppID: merchantAppID,
		TradeNo:       tradeNo,
		OutTradeNo:    merchantTradeNo,
		ChannelType:   payChannel.ChannelType,
		Amount:        amount,
		Subject:       subject,
		Status:        model.OrderStatusPending,
		NotifyURL:     notifyURL,
		Extra:         "{}",
	}
	if err := s.store.DB.Create(&order).Error; err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// 5. 根据渠道向微信或支付宝服务端下单
	switch channelType {
	case string(model.ChannelTypeWechat):
		return s.createWechatOrder(ctx, &order, &payChannel)
	case string(model.ChannelTypeAlipay):
		return s.createAlipayOrder(ctx, &order, &payChannel)
	default:
		return nil, fmt.Errorf("unsupported channel type: %s", channelType)
	}
}

// createWechatOrder 微信 V3 下单（此处示例 Native 支付）。
func (s *PayService) createWechatOrder(ctx context.Context, order *model.Order, channel *model.PayChannel) (*UnifiedOrderResult, error) {
	// 初始化 ClientV3（从 channel 直接读取已解密的 PrivateKey / APIv3Key）
	client, err := wechat.NewClientV3(channel.MchID, channel.SerialNo, channel.APIv3Key, channel.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to init wechat client: %w", err)
	}

	notifyUrl := order.NotifyURL
	if notifyUrl == "" {
		notifyUrl = "https://paygate.example.com/api/v1/pay/notify/wechat"
	}

	bm := make(gopay.BodyMap)
	bm.Set("appid", channel.AppID).
		Set("mchid", channel.MchID).
		Set("description", order.Subject).
		Set("out_trade_no", order.TradeNo).
		Set("notify_url", notifyUrl)
	bm.SetBodyMap("amount", func(b gopay.BodyMap) {
		b.Set("total", order.Amount).Set("currency", "CNY")
	})

	// 请求 Native 下单 API
	wxRsp, err := client.V3TransactionNative(ctx, bm)
	if err != nil {
		return nil, fmt.Errorf("wechat native api failed: %w", err)
	}
	if wxRsp.Code != wechat.Success {
		return nil, fmt.Errorf("wechat api error: %s - %s", wxRsp.Error, wxRsp.Error)
	}

	// 微信 Native 支付返回的是一个 code_url，商户可将其转为二维码让用户扫
	return &UnifiedOrderResult{
		TradeNo: order.TradeNo,
		PayParams: map[string]string{
			"code_url": wxRsp.Response.CodeUrl,
		},
	}, nil
}
