package controller

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"paygate-omni/internal/model"
)

// GetStats 获取概览数据
func (c *AdminController) GetStats(ctx *gin.Context) {
	var mCount, cCount, oCount int64
	c.db.Model(&model.Merchant{}).Count(&mCount)
	c.db.Model(&model.PayChannel{}).Count(&cCount)
	c.db.Model(&model.Order{}).Count(&oCount)

	type Result struct {
		Total int64
	}
	var totalAmount Result
	c.db.Model(&model.Order{}).Where("status = ?", model.OrderStatusSuccess).Select("COALESCE(SUM(amount), 0) as total").Scan(&totalAmount)

	var todayAmount Result
	todayStr := time.Now().Format("2006-01-02")
	c.db.Model(&model.Order{}).Where("status = ? AND created_at >= ?", model.OrderStatusSuccess, todayStr).Select("COALESCE(SUM(amount), 0) as total").Scan(&todayAmount)

	ctx.JSON(http.StatusOK, gin.H{
		"code": "SUCCESS",
		"data": gin.H{
			"merchant_count": mCount,
			"channel_count":  cCount,
			"order_count":    oCount,
			"total_amount":   totalAmount.Total,
			"today_amount":   todayAmount.Total,
		},
	})
}
