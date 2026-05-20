package controller

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"paygate-omni/internal/model"
	"paygate-omni/internal/service"
)

// EpayController 提供易支付兼容接口。
type EpayController struct {
	logger *zap.Logger
	db     *gorm.DB
	paySvc *service.PayService
}

// NewEpayController 创建兼容控制器。
func NewEpayController(logger *zap.Logger, db *gorm.DB, paySvc *service.PayService) *EpayController {
	return &EpayController{logger: logger, db: db, paySvc: paySvc}
}

// EpayPayRequest 表示易支付常见下单参数。
type EpayPayRequest struct {
	Pid        string `form:"pid" json:"pid" binding:"required"`
	Type       string `form:"type" json:"type" binding:"required,oneof=alipay wechat wxpay"`
	OutTradeNo string `form:"out_trade_no" json:"out_trade_no" binding:"required,max=64"`
	NotifyURL  string `form:"notify_url" json:"notify_url" binding:"required,url"`
	ReturnURL  string `form:"return_url" json:"return_url" binding:"omitempty,url"`
	Name       string `form:"name" json:"name" binding:"required,max=256"`
	Money      string `form:"money" json:"money" binding:"required"`
	Sign       string `form:"sign" json:"sign" binding:"required"`
	SignType   string `form:"sign_type" json:"sign_type"`
}

// EpayQueryRequest 表示易支付常见查询参数。
type EpayQueryRequest struct {
	Pid        string `form:"pid" json:"pid" binding:"required"`
	OutTradeNo string `form:"out_trade_no" json:"out_trade_no" binding:"required"`
	Sign       string `form:"sign" json:"sign" binding:"required"`
	SignType   string `form:"sign_type" json:"sign_type"`
}

// Submit 兼容易支付提交接口。
func (e *EpayController) Submit(c *gin.Context) {
	e.handlePayInternal(c, true)
}

// API 兼容易支付 api.php 入口。
func (e *EpayController) API(c *gin.Context) {
	act := strings.ToLower(strings.TrimSpace(c.Query("act")))
	switch act {
	case "pay", "submit":
		e.handlePayInternal(c, false)
	case "query", "order":
		e.Query(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 0, "msg": "unsupported act"})
	}
}

func (e *EpayController) handlePayInternal(c *gin.Context, isHTML bool) {
	renderError := func(msg string, code int) {
		if isHTML {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(code, "<h2>支付失败: %s</h2>", msg)
		} else {
			c.JSON(code, gin.H{"code": 0, "msg": msg})
		}
	}
	var req EpayPayRequest
	if err := c.ShouldBind(&req); err != nil {
		renderError(err.Error(), http.StatusBadRequest)
		return
	}

	values, err := requestValues(c)
	if err != nil {
		renderError(err.Error(), http.StatusBadRequest)
		return
	}

	merchantAppID, secretKey, err := e.resolveMerchant(req.Pid)
	if err != nil {
		renderError(err.Error(), http.StatusUnauthorized)
		return
	}

	if !verifyEpaySign(values, secretKey) {
		e.logger.Warn("epay compat: signature mismatch", zap.String("pid", req.Pid), zap.String("out_trade_no", req.OutTradeNo))
		renderError("签名校验失败", http.StatusUnauthorized)
		return
	}

	amountFen, err := moneyToFen(req.Money)
	if err != nil {
		renderError(err.Error(), http.StatusBadRequest)
		return
	}

	channelType := string(model.ChannelTypeAlipay)
	if req.Type == "wechat" || req.Type == "wxpay" {
		channelType = string(model.ChannelTypeWechat)
	}

	res, err := e.paySvc.CreateOrder(c.Request.Context(), merchantAppID, req.OutTradeNo, req.Name, channelType, req.NotifyURL, amountFen)
	if err != nil {
		e.logger.Error("epay compat: create order failed", zap.Error(err), zap.String("pid", req.Pid), zap.String("out_trade_no", req.OutTradeNo))
		renderError(err.Error(), http.StatusOK)
		return
	}

	payURL := buildPayURL(res)
	if isHTML {
		// 返回收银台页面展示二维码
		c.Header("Content-Type", "text/html; charset=utf-8")
		title := "支付宝"
		if channelType == string(model.ChannelTypeWechat) {
			title = "微信"
		}
		queryValues := url.Values{}
		queryValues.Set("act", "query")
		queryValues.Set("out_trade_no", req.OutTradeNo)
		queryValues.Set("pid", req.Pid)
		querySign := generateEpaySign(queryValues, secretKey)

		html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="utf-8">
			<title>PayGate 聚合收银台</title>
			<meta name="viewport" content="width=device-width, initial-scale=1">
			<style>
				body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; text-align: center; margin: 0; padding-top: 50px; background: #f0f2f5; }
				.card { background: white; max-width: 360px; margin: 0 auto; padding: 40px 20px; border-radius: 12px; box-shadow: 0 8px 24px rgba(0,0,0,0.08); }
				h2 { color: #333; margin-top: 0; font-size: 20px;}
				.money { color: #f56c6c; font-size: 32px; margin: 20px 0; font-weight: 600; }
				.money span { font-size: 20px; margin-right: 2px;}
				.qr-wrapper { margin: 20px auto; padding: 12px; background: #fff; border: 1px solid #ebedf0; display: inline-block; border-radius: 8px;}
				.tip { color: #666; font-size: 14px; margin-bottom: 5px; }
			</style>
		</head>
		<body>
			<div class="card">
				<h2 id="title">%s扫码支付</h2>
				<div class="money"><span>￥</span>%s</div>
				<div class="qr-wrapper" id="qr-wrapper">
					<img src="https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=%s&margin=1" alt="QR Code" width="220" height="220">
				</div>
				<div class="tip" id="tip">请打开%s扫一扫，扫描二维码完成支付</div>
				<div class="tip" style="margin-top:30px; font-size:12px; color:#aaa;">订单号: %s</div>
			</div>
			<script>
				setInterval(function() {
					fetch('/api.php?act=query&out_trade_no=%s&pid=%s&sign=%s')
					.then(res => res.json())
					.then(data => {
						if (data.code === 1) {
							document.getElementById('title').innerText = '支付成功';
							document.getElementById('qr-wrapper').innerHTML = '<div style="color: #67c23a; font-size: 60px;">✅</div>';
							document.getElementById('tip').innerText = '支付已完成，请返回原网站';
						}
					}).catch(e => console.error(e));
				}, 3000);
			</script>
		</body>
		</html>`, title, req.Money, url.QueryEscape(payURL), title, res.TradeNo, req.OutTradeNo, req.Pid, querySign)
		c.String(http.StatusOK, html)
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "success",
			"data": gin.H{
				"pid":          req.Pid,
				"trade_no":     res.TradeNo,
				"out_trade_no": req.OutTradeNo,
				"money":        req.Money,
				"type":         req.Type,
				"notify_url":   req.NotifyURL,
				"pay_url":      payURL,
				"pay_params":   res.PayParams,
			},
		})
	}
}

// Query 兼容易支付订单查询接口。
func (e *EpayController) Query(c *gin.Context) {
	var req EpayQueryRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 0, "msg": err.Error()})
		return
	}

	values, err := requestValues(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 0, "msg": err.Error()})
		return
	}

	_, secretKey, err := e.resolveMerchant(req.Pid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 0, "msg": err.Error()})
		return
	}

	if !verifyEpaySign(values, secretKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 0, "msg": "签名校验失败"})
		return
	}

	var order model.Order
	if err := e.db.Where("merchant_app_id = ? AND out_trade_no = ?", req.Pid, req.OutTradeNo).First(&order).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "order not found"})
		return
	}

	statusCode := 0
	if order.Status == model.OrderStatusSuccess {
		statusCode = 1
	}

	c.JSON(http.StatusOK, gin.H{
		"code": statusCode,
		"msg":  "success",
		"data": gin.H{
			"pid":          req.Pid,
			"trade_no":     order.TradeNo,
			"out_trade_no": order.OutTradeNo,
			"money":        fmt.Sprintf("%.2f", float64(order.Amount)/100),
			"type":         string(order.ChannelType),
			"status":       string(order.Status),
		},
	})
}

// resolveMerchant 通过 pid 获取商户 AppID 和密钥。
func (e *EpayController) resolveMerchant(pid string) (string, string, error) {
	if pid == "" {
		return "", "", fmt.Errorf("pid is required")
	}

	var merchant model.Merchant
	if err := e.db.Where("app_id = ? AND is_active = true", pid).First(&merchant).Error; err != nil {
		return "", "", fmt.Errorf("merchant not found")
	}
	return merchant.AppID, merchant.SecretKey, nil
}

// requestValues 读取请求参数，兼容 query 和 form 两种提交方式。
func requestValues(c *gin.Context) (url.Values, error) {
	if err := c.Request.ParseForm(); err != nil {
		return nil, fmt.Errorf("invalid request parameters")
	}
	values := make(url.Values, len(c.Request.Form))
	for key, list := range c.Request.Form {
		if len(list) == 0 {
			continue
		}
		values[key] = []string{list[0]}
	}
	return values, nil
}

// moneyToFen 将元金额转换为分。
func moneyToFen(money string) (int64, error) {
	parts := strings.SplitN(strings.TrimSpace(money), ".", 3)
	if len(parts) == 1 {
		v, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid money")
		}
		return v * 100, nil
	}
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid money")
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid money")
	}
	frac := parts[1]
	if len(frac) == 1 {
		frac += "0"
	}
	if len(frac) > 2 {
		frac = frac[:2]
	}
	decimal, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid money")
	}
	return whole*100 + decimal, nil
}

// generateEpaySign 按照易支付规则生成MD5签名
func generateEpaySign(values url.Values, secretKey string) string {
	keys := make([]string, 0, len(values))
	for key, list := range values {
		if key == "sign" || key == "sign_type" || key == "act" || len(list) == 0 || list[0] == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	payload := strings.Join(parts, "&") + secretKey
	sum := md5.Sum([]byte(payload))
	return hex.EncodeToString(sum[:])
}

// verifyEpaySign 按易支付常见规则校验 MD5 签名。
func verifyEpaySign(values url.Values, secretKey string) bool {
	signType := strings.ToUpper(strings.TrimSpace(values.Get("sign_type")))
	if signType != "" && signType != "MD5" {
		return false
	}

	keys := make([]string, 0, len(values))
	for key, list := range values {
		if key == "sign" || key == "sign_type" || len(list) == 0 || list[0] == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	payload := strings.Join(parts, "&") + secretKey
	sum := md5.Sum([]byte(payload))
	return strings.EqualFold(hex.EncodeToString(sum[:]), values.Get("sign"))
}

// buildPayURL 从统一下单结果中提取可跳转或展示的支付链接。
func buildPayURL(res *service.UnifiedOrderResult) string {
	if res == nil || res.PayParams == nil {
		return ""
	}
	if params, ok := res.PayParams.(map[string]string); ok {
		if codeURL, exists := params["code_url"]; exists {
			return codeURL
		}
		if qrCode, exists := params["qr_code"]; exists {
			return qrCode
		}
	}
	return ""
}
