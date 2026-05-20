import sys
from pathlib import Path

p = Path('/home/ch/epay/backend/internal/controller/epay.go')
content = p.read_text()

old_html = '''			<div class="card">
				<h2>%s扫码支付</h2>
				<div class="money"><span>￥</span>%s</div>
				<div class="qr-wrapper">
					<img src="https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=%s&margin=1" alt="QR Code" width="220" height="220">
				</div>
				<div class="tip">请打开%s扫一扫，扫描二维码完成支付</div>
				<div class="tip" style="margin-top:30px; font-size:12px; color:#aaa;">订单号: %s</div>
			</div>
		</body>'''

new_html = '''			<div class="card">
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
		</body>'''

# We need to generate the sign for the query, or we can just skip sign checking if it's internal...? No, query requires sign!
# Wait, we can generate the sign for query!

old_render = '''html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>'''

new_render = '''queryValues := url.Values{}
		queryValues.Set("act", "query")
		queryValues.Set("out_trade_no", req.OutTradeNo)
		queryValues.Set("pid", req.Pid)
		querySign := generateEpaySign(queryValues, secretKey)

		html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>'''

old_html_invoke = '''</html>`, title, req.Money, url.QueryEscape(payURL), title, res.TradeNo)'''
new_html_invoke = '''</html>`, title, req.Money, url.QueryEscape(payURL), title, res.TradeNo, req.OutTradeNo, req.Pid, querySign)'''

if old_html not in content:
    sys.exit('missing old_html')
if old_render not in content:
    sys.exit('missing old_render')
if old_html_invoke not in content:
    sys.exit('missing old_html_invoke')

content = content.replace(old_html, new_html)
content = content.replace(old_render, new_render)
content = content.replace(old_html_invoke, new_html_invoke)

sign_algo = '''// generateEpaySign 按照易支付规则生成MD5签名
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

// verifyEpaySign'''

content = content.replace('// verifyEpaySign', sign_algo)

p.write_text(content)
print('inject polling JS to HTML')
