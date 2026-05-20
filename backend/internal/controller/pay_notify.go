package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (p *PayController) NotifyPay(c *gin.Context) {
	channel := c.Param("channel")
	p.logger.Info("pay notify: received callback", zap.String("channel", channel), zap.String("ip", c.ClientIP()))

	switch channel {
	case "wechat":
		err := p.paySvc.HandleWechatNotify(c.Request)
		if err != nil {
			p.logger.Error("wechat notify handle failed", zap.Error(err))
			c.XML(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "Handle failed"})
			return
		}
		c.XML(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})

	case "alipay":
		err := p.paySvc.HandleAlipayNotify(c.Request)
		if err != nil {
			p.logger.Error("alipay notify handle failed", zap.Error(err))
			c.String(http.StatusBadRequest, "fail")
			return
		}
		c.String(http.StatusOK, "success")

	default:
		p.logger.Warn("pay notify: unknown channel", zap.String("channel", channel))
		c.JSON(http.StatusBadRequest, gin.H{"code": "UNKNOWN_CHANNEL", "message": "Unsupported channel"})
	}
}
