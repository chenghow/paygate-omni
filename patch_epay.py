import sys
from pathlib import Path

p = Path('/home/ch/epay/backend/internal/controller/epay.go')
content = p.read_text()

# 1. Replace Submit and HandlePay definitions
old_submit_api = '''// Submit 兼容易支付提交接口。
func (e *EpayController) Submit(c *gin.Context) {
	e.HandlePay(c)
}

// API 兼容易支付 api.php 入口。
func (e *EpayController) API(c *gin.Context) {
	act := strings.ToLower(strings.TrimSpace(c.Query("act")))
	switch act {
	case "pay", "submit":
		e.HandlePay(c)
	case "query", "order":
		e.Query(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 0, "msg": "unsupported act"})
	}
}

// HandlePay 兼容易支付下单接口。
func (e *EpayController) HandlePay(c *gin.Context) {'''

new_submit_api = '''// Submit 兼容易支付提交接口。
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
	}'''

if old_submit_api not in content:
    sys.exit('Could not find old_submit_api block')
content = content.replace(old_submit_api, new_submit_api)

# 2. Replace all basic JSON error returns with renderError
content = content.replace(
'''	var req EpayPayRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 0, "msg": err.Error()})
		return
	}''',
'''	var req EpayPayRequest
	if err := c.ShouldBind(&req); err != nil {
		renderError(err.Error(), http.StatusBadRequest)
		return
	}''')

content = content.replace(
'''	values, err := requestValues(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 0, "msg": err.Error()})
		return
	}''',
'''	values, err := requestValues(c)
	if err != nil {
		renderError(err.Error(), http.StatusBadRequest)
		return
	}''')

content = content.replace(
'''	merchantAppID, secretKey, err := e.resolveMerchant(req.Pid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 0, "msg": err.Error()})
		return
	}''',
'''	merchantAppID, secretKey, err := e.resolveMerchant(req.Pid)
	if err != nil {
		renderError(err.Error(), http.StatusUnauthorized)
		return
	}''')

content = content.replace(
'''	if !verifyEpaySign(values, secretKey) {
		e.logger.Warn("epay compat: signature mismatch", zap.String("pid", req.Pid), zap.String("out_trade_no", req.OutTradeNo))
		c.JSON(http.StatusUnauthorized, gin.H{"code": 0, "msg": "签名校验失败"})
		return
	}''',
'''	if !verifyEpaySign(values, secretKey) {
		e.logger.Warn("epay compat: signature mismatch", zap.String("pid", req.Pid), zap.String("out_trade_no", req.OutTradeNo))
		renderError("签名校验失败", http.StatusUnauthorized)
		return
	}''')

content = content.replace(
'''	amountFen, err := moneyToFen(req.Money)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 0, "msg": err.Error()})
		return
	}''',
'''	amountFen, err := moneyToFen(req.Money)
	if err != nil {
		renderError(err.Error(), http.StatusBadRequest)
		return
	}''')

content = content.replace(
'''	res, err := e.paySvc.CreateOrder(c.Request.Context(), merchantAppID, req.OutTradeNo, req.Name, channelType, req.NotifyURL, amountFen)
	if err != nil {
		e.logger.Error("epay compat: create order failed", zap.Error(err), zap.String("pid", req.Pid), zap.String("out_trade_no", req.OutTradeNo))
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": err.Error()})
		return
	}''',
'''	res, err := e.paySvc.CreateOrder(c.Request.Context(), merchantAppID, req.OutTradeNo, req.Name, channelType, req.NotifyURL, amountFen)
	if err != nil {
		e.logger.Error("epay compat: create order failed", zap.Error(err), zap.String("pid", req.Pid), zap.String("out_trade_no", req.OutTradeNo))
		renderError(err.Error(), http.StatusOK)
		return
	}''')

# 3. Replace success JSON response with the dual-mode response
old_success = '''	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"msg":  "success",
		"data": gin.H{
			"pid":          req.Pid,
			"trade_no":     res.TradeNo,
			"out_trade_no": req.OutTradeNo,
			"money":        req.Money,
			"type":         req.Type,
			"notify_url":   req.NotifyURL,
			"pay_url":      buildPayURL(res),
			"pay_params":   res.PayParams,
		},
	})
}'''
new_success = '''	payURL := buildPayURL(res)
	if isHTML {
		// 返回收银台页面展示二维码
		c.Header("Content-Type", "text/html; charset=utf-8")
		title := "支付宝"
		if channelType == string(model.ChannelTypeWechat) {
			title = "微信"
		}
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
				<h2>%s扫码支付</h2>
				<div class="money"><span>￥</span>%s</div>
				<div class="qr-wrapper">
					<img src="https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=%s&margin=1" alt="QR Code" width="220" height="220">
				</div>
				<div class="tip">请打开%s扫一扫，扫描二维码完成支付</div>
				<div class="tip" style="margin-top:30px; font-size:12px; color:#aaa;">订单号: %s</div>
			</div>
		</body>
		</html>`, title, req.Money, url.QueryEscape(payURL), title, res.TradeNo)
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
}'''

if old_success not in content:
    sys.exit('Could not find success JSON block')

content = content.replace(old_success, new_success)
p.write_text(content)
print("HTML rendering added successfully")
