package controller

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	
	"paygate-omni/internal/service"
)

type PayController struct {
	logger  *zap.Logger
	paySvc  *service.PayService
}

func NewPayController(logger *zap.Logger, paySvc *service.PayService) *PayController {
	return &PayController{logger: logger, paySvc: paySvc}
}

type CreatePayRequest struct {
	MerchantTradeNo string `json:"merchant_trade_no" binding:"required,max=64"`
	Amount          int64  `json:"amount"            binding:"required,min=1"`
	Subject         string `json:"subject"           binding:"required,max=256"`
	ChannelType     string `json:"channel_type"      binding:"required,oneof=wechat alipay"`
	NotifyURL       string `json:"notify_url"        binding:"required,url"`
}

type CreatePayResponse struct {
	TradeNo   string      `json:"trade_no"`
	PayParams interface{} `json:"pay_params"`
}

func (p *PayController) CreatePay(c *gin.Context) {
	appID, _ := c.Get("merchant_app_id")

	var req CreatePayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		p.logger.Warn("pay create: invalid request params", zap.Any("merchant_app_id", appID), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_PARAMS", "message": err.Error()})
		return
	}

	p.logger.Info("pay create: received order request",
		zap.Any("merchant_app_id", appID),
		zap.String("merchant_trade_no", req.MerchantTradeNo),
		zap.Int64("amount", req.Amount),
		zap.String("channel_type", req.ChannelType),
		zap.String("subject", req.Subject),
	)

	// 调用 Service 完成订单创建与发起
	appIDStr, _ := appID.(string)
	res, err := p.paySvc.CreateOrder(context.Background(), appIDStr, req.MerchantTradeNo, req.Subject, req.ChannelType, req.NotifyURL, req.Amount)
	if err != nil {
		p.logger.Error("pay create failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": "SUCCESS",
		"message": "Order created successfully",
		"data": CreatePayResponse{
			TradeNo:   res.TradeNo,
			PayParams: res.PayParams,
		},
	})
}

