package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"

	"fmt"
	"net/http"
	"time"

	"github.com/go-pay/gopay/alipay"
	"github.com/go-pay/gopay/wechat/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"paygate-omni/internal/model"
)

// HandleWechatNotify 解析微信支付异步通知并处理订单状态流转 (加锁+变库)
func (s *PayService) HandleWechatNotify(req *http.Request) error {
	notifyReq, err := wechat.V3ParseNotify(req)
	if err != nil {
		return fmt.Errorf("failed to parse wechat notify request: %w", err)
	}

	var channel model.PayChannel
	if err := s.store.DB.Where("channel_type = ? AND is_active = true", "wechat").First(&channel).Error; err != nil {
		return fmt.Errorf("could not find active wechat channel: %w", err)
	}

	// 3. 直接拿 APIV3Key 解密 resource
	result, err := notifyReq.DecryptPayCipherText(channel.APIv3Key)
	if err != nil {
		return fmt.Errorf("failed to decrypt wechat notify: %w", err)
	}

	tradeNo := result.OutTradeNo
	s.logger.Info("parsed wechat notify", zap.String("trade_no", tradeNo), zap.String("trade_state", result.TradeState))

	lockKey := "pay_lock:wechat:" + tradeNo
	ok, err := s.store.RDB.SetNX(context.Background(), lockKey, "locked", 10*time.Second).Result()
	if err != nil || !ok {
		return fmt.Errorf("concurrent modification or get lock failed for tradeNo: %s", tradeNo)
	}
	defer s.store.RDB.Del(context.Background(), lockKey)

	return s.store.DB.Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Where("trade_no = ?", tradeNo).First(&order).Error; err != nil {
			return err
		}

		if order.Status == "SUCCESS" || order.Status == "FAILED" {
			return nil
		}

		if result.TradeState == "SUCCESS" {
			order.Status = "SUCCESS"
		} else if result.TradeState == "PAYERROR" {
			order.Status = "FAILED"
		} else if result.TradeState == "CLOSED" {
			order.Status = "CLOSED"
		} else {
			return nil
		}

		if err := tx.Save(&order).Error; err != nil {
			return err
		}

		if order.Status == "SUCCESS" {
			go s.asyncNotifyMerchant(&order)
		}

		return nil
	})
}

// HandleAlipayNotify 解析支付宝异步通知并处理订单状态流转 (加锁+变库)
func (s *PayService) HandleAlipayNotify(req *http.Request) error {
	bm, err := alipay.ParseNotifyToBodyMap(req)
	if err != nil {
		return fmt.Errorf("failed to parse alipay notify request: %w", err)
	}

	tradeNo := bm.GetString("out_trade_no")
	appID := bm.GetString("app_id")
	tradeStatus := bm.GetString("trade_status")

	s.logger.Info("parsed alipay notify", zap.String("trade_no", tradeNo), zap.String("trade_status", tradeStatus))

	var channel model.PayChannel
	if err := s.store.DB.Where("app_id = ? AND channel_type = ? AND is_active = true", appID, "alipay").First(&channel).Error; err != nil {
		return fmt.Errorf("could not find active alipay channel for appID %s: %w", appID, err)
	}

	ok, err := alipay.VerifySign(channel.AlipayPublicKey, bm)
	if err != nil || !ok {
		return fmt.Errorf("failed to verify alipay signature, ok=%v, err=%v", ok, err)
	}

	lockKey := "pay_lock:alipay:" + tradeNo
	lockOk, err := s.store.RDB.SetNX(context.Background(), lockKey, "locked", 10*time.Second).Result()
	if err != nil || !lockOk {
		return fmt.Errorf("concurrent modification or get lock failed for tradeNo: %s", tradeNo)
	}
	defer s.store.RDB.Del(context.Background(), lockKey)

	return s.store.DB.Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Where("trade_no = ?", tradeNo).First(&order).Error; err != nil {
			return err
		}

		if order.Status == "SUCCESS" || order.Status == "FAILED" {
			return nil
		}

		if tradeStatus == "TRADE_SUCCESS" || tradeStatus == "TRADE_FINISHED" {
			order.Status = "SUCCESS"
		} else if tradeStatus == "TRADE_CLOSED" {
			order.Status = "CLOSED"
		} else {
			return nil
		}

		if err := tx.Save(&order).Error; err != nil {
			return err
		}

		if order.Status == "SUCCESS" {
			go s.asyncNotifyMerchant(&order)
		}

		return nil
	})
}

func (s *PayService) asyncNotifyMerchant(order *model.Order) {
	s.logger.Info("trigger notify to merchant", zap.String("merchant", order.MerchantAppID), zap.String("url", order.NotifyURL))

	if order.NotifyURL == "" {
		return
	}

	var merchant model.Merchant
	if err := s.store.DB.Where("app_id = ?", order.MerchantAppID).First(&merchant).Error; err != nil {
		s.logger.Error("notify failed: merchant not found", zap.Error(err))
		return
	}

	values := url.Values{}
	values.Set("pid", order.MerchantAppID)
	values.Set("trade_no", order.TradeNo)
	values.Set("out_trade_no", order.OutTradeNo)

	// Convert type
	chType := string(order.ChannelType)
	if chType == "wechat" {
		chType = "wxpay"
	}
	values.Set("type", chType)
	values.Set("name", order.Subject)
	values.Set("money", fmt.Sprintf("%.2f", float64(order.Amount)/100.0))
	values.Set("trade_status", "TRADE_SUCCESS")

	// Generate Sign
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var parts []string
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	payload := strings.Join(parts, "&") + merchant.SecretKey

	sum := md5.Sum([]byte(payload))
	sign := hex.EncodeToString(sum[:])
	values.Set("sign", sign)
	values.Set("sign_type", "MD5")

	notifyURL, err := url.Parse(order.NotifyURL)
	if err != nil {
		s.logger.Error("invalid notify url", zap.Error(err))
		return
	}
	query := notifyURL.Query()
	for k, v := range values {
		query[k] = v
	}
	notifyURL.RawQuery = query.Encode()

	resp, err := http.Get(notifyURL.String())
	if err != nil {
		s.logger.Error("send notify request failed", zap.Error(err))
		return
	}
	defer resp.Body.Close()
	s.logger.Info("sent notify to merchant", zap.String("url", notifyURL.String()), zap.Int("status", resp.StatusCode))
}
