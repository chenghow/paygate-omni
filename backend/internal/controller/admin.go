package controller

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"paygate-omni/config"
	"paygate-omni/internal/model"
)

// AdminController 处理管理后台相关请求。
type AdminController struct {
	logger *zap.Logger
	cfg    *config.Config
	db     *gorm.DB
	rdb    *redis.Client
}

func NewAdminController(logger *zap.Logger, cfg *config.Config, db *gorm.DB, rdb *redis.Client) *AdminController {
	return &AdminController{logger: logger, cfg: cfg, db: db, rdb: rdb}
}

type AdminLoginReq struct {
	Password string `json:"password" binding:"required"`
}

// Login 管理员登录
func (c *AdminController) Login(ctx *gin.Context) {
	var req AdminLoginReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_PARAMS", "message": err.Error()})
		return
	}

	if req.Password != c.cfg.Security.AdminPassword {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "invalid password"})
		return
	}

	// 生成 Token
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		c.logger.Error("failed to generate admin token", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR"})
		return
	}
	token := hex.EncodeToString(b)

	// 存入 Redis，24小时过期
	err := c.rdb.Set(context.Background(), "admin_session:"+token, "ok", 24*time.Hour).Err()
	if err != nil {
		c.logger.Error("failed to save admin session", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "token": token})
}


// --- 商户管理 CRUD ---

type MerchantInput struct {
	Name      string `json:"name"`
	AppID     string `json:"app_id"`
	SecretKey string `json:"secret_key"`
	IsActive  bool   `json:"is_active"`
}

type MerchantListItem struct {
	model.Merchant
	TotalVolume int64 `json:"total_volume"`
}

func (c *AdminController) ListMerchants(ctx *gin.Context) {
	var merchants []model.Merchant
	if err := c.db.Find(&merchants).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "DB_ERROR", "message": err.Error()})
		return
	}

	var results []MerchantListItem
	for _, m := range merchants {
		var total int64
		c.db.Model(&model.Order{}).
			Where("merchant_app_id = ? AND status = ?", m.AppID, model.OrderStatusSuccess).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&total)

		results = append(results, MerchantListItem{
			Merchant:    m,
			TotalVolume: total,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "data": results})
}

func (c *AdminController) CreateMerchant(ctx *gin.Context) {
	var req MerchantInput
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_PARAMS"})
		return
	}
	m := model.Merchant{
		Name:      req.Name,
		AppID:     req.AppID,
		SecretKey: req.SecretKey, // Assign to struct explicitly so BeforeSave encrypts it
		IsActive:  req.IsActive,
	}
	if err := c.db.Create(&m).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "DB_ERROR", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "data": m})
}

func (c *AdminController) UpdateMerchant(ctx *gin.Context) {
	id := ctx.Param("id")
	var m model.Merchant
	if err := c.db.First(&m, id).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND"})
		return
	}
	var req MerchantInput
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_PARAMS"})
		return
	}
	m.Name = req.Name
	m.IsActive = req.IsActive
	if req.SecretKey != "" {
		m.SecretKey = req.SecretKey // Only update if provided
	}
	if err := c.db.Save(&m).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "DB_ERROR"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": "SUCCESS"})
}

func (c *AdminController) DeleteMerchant(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := c.db.Delete(&model.Merchant{}, id).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "DB_ERROR"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": "SUCCESS"})
}

// --- 渠道配置 CRUD ---

type ChannelInput struct {
	MerchantAppID   string            `json:"merchant_app_id"`
	ChannelType     model.ChannelType `json:"channel_type"`
	AppID           string            `json:"app_id"`
	MchID           string            `json:"mch_id"`
	SerialNo        string            `json:"serial_no"`
	APIv3Key        string            `json:"api_v3_key"`
	PrivateKey      string            `json:"private_key"`
	AlipayPublicKey string            `json:"alipay_public_key"`
	IsSandbox       bool              `json:"is_sandbox"`
	IsActive        bool              `json:"is_active"`
}

func (c *AdminController) ListChannels(ctx *gin.Context) {
	var channels []model.PayChannel
	if err := c.db.Find(&channels).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "DB_ERROR"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "data": channels})
}

func (c *AdminController) CreateChannel(ctx *gin.Context) {
	var req ChannelInput
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_PARAMS"})
		return
	}
	ch := model.PayChannel{
		MerchantAppID:   req.MerchantAppID,
		ChannelType:     req.ChannelType,
		AppID:           req.AppID,
		MchID:           req.MchID,
		SerialNo:        req.SerialNo,
		APIv3Key:        req.APIv3Key,
		PrivateKey:      req.PrivateKey,
		AlipayPublicKey: req.AlipayPublicKey,
		IsSandbox:       req.IsSandbox,
		IsActive:        req.IsActive,
	}
	if err := c.db.Create(&ch).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "DB_ERROR", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "data": ch})
}

func (c *AdminController) UpdateChannel(ctx *gin.Context) {
	id := ctx.Param("id")
	var ch model.PayChannel
	if err := c.db.First(&ch, id).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND"})
		return
	}
	var req ChannelInput
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_PARAMS"})
		return
	}

	ch.MerchantAppID = req.MerchantAppID
	ch.ChannelType = req.ChannelType
	ch.AppID = req.AppID
	ch.MchID = req.MchID
	ch.SerialNo = req.SerialNo
	ch.IsSandbox = req.IsSandbox
	ch.IsActive = req.IsActive

	if req.APIv3Key != "" {
		ch.APIv3Key = req.APIv3Key
	}
	if req.PrivateKey != "" {
		ch.PrivateKey = req.PrivateKey
	}
	if req.AlipayPublicKey != "" {
		ch.AlipayPublicKey = req.AlipayPublicKey
	}

	if err := c.db.Save(&ch).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "DB_ERROR"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": "SUCCESS"})
}

func (c *AdminController) DeleteChannel(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := c.db.Delete(&model.PayChannel{}, id).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "DB_ERROR"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": "SUCCESS"})
}

func (c *AdminController) ListOrders(ctx *gin.Context) {
	var orders []model.Order
	if err := c.db.Order("created_at desc").Find(&orders).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": "DB_ERROR", "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "data": orders})
}
