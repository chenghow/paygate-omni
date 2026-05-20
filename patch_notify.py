import sys
from pathlib import Path

p = Path('/home/ch/epay/backend/internal/service/pay_notify.go')
content = p.read_text()

old = '''func (s *PayService) asyncNotifyMerchant(order *model.Order) {
	s.logger.Info("trigger notify to merchant", zap.String("merchant", order.MerchantAppID), zap.String("url", order.NotifyURL))
}'''

new = '''import_crypto_md5 := "crypto/md5"
import_encoding_hex := "encoding/hex"
import_net_url := "net/url"
import_sort := "sort"
import_strings := "strings"

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
}'''

new = new.replace("import_crypto_md5 := \"crypto/md5\"", "")
new = new.replace("import_encoding_hex := \"encoding/hex\"", "")
new = new.replace("import_net_url := \"net/url\"", "")
new = new.replace("import_sort := \"sort\"", "")
new = new.replace("import_strings := \"strings\"", "")

if old not in content:
    sys.exit('old implementation not found')

content = content.replace(old, new)
p.write_text(content)
print("async notify overwritten")
