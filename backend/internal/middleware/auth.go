package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	headerAppID     = "X-Pay-AppID"
	headerTimestamp = "X-Pay-Timestamp"
	headerNonce     = "X-Pay-Nonce"
	headerSignature = "X-Pay-Signature"

	maxTimestampDrift = 5 * time.Minute
	nonceTTL          = 5 * time.Minute
	nonceLen          = 16
	maxBodySize       = 1 << 20 // 1MB
)

// SecretKeyFn 是从商户 AppID 查询其明文 SecretKey 的函数类型。
type SecretKeyFn func(appID string) (string, error)

// SignatureVerifier 返回面向商户侧接口的签名验证 + 防重放中间件。
//
// 校验流程（AGENTS.md §4.2）：
//  1. 检查必要 Header 是否齐全
//  2. 校验 Timestamp 时效（±5 分钟）
//  3. Redis SetNX 检测 Nonce 是否重用（防重放）
//  4. 读取请求体，计算 HMAC-SHA256 与 Signature 时序安全比较
func SignatureVerifier(secretKeyFn SecretKeyFn, rdb *redis.Client, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		appID := c.GetHeader(headerAppID)
		tsStr := c.GetHeader(headerTimestamp)
		nonce := c.GetHeader(headerNonce)
		signature := c.GetHeader(headerSignature)

		// Step 1：Header 完整性检查
		if appID == "" || tsStr == "" || nonce == "" || signature == "" {
			logger.Warn("auth: missing required headers",
				zap.String("ip", c.ClientIP()),
				zap.String("app_id", appID),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "MISSING_AUTH_HEADERS",
				"message": "X-Pay-AppID, X-Pay-Timestamp, X-Pay-Nonce, X-Pay-Signature are all required",
			})
			return
		}

		// Step 2：Timestamp 时效校验
		tsUnix, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "INVALID_TIMESTAMP",
				"message": "X-Pay-Timestamp must be a Unix timestamp integer",
			})
			return
		}
		drift := time.Since(time.Unix(tsUnix, 0))
		if drift < 0 {
			drift = -drift
		}
		if drift > maxTimestampDrift {
			logger.Warn("auth: timestamp drift exceeded",
				zap.String("app_id", appID),
				zap.Duration("drift", drift),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "TIMESTAMP_EXPIRED",
				"message": "Request timestamp is too old or too far in the future",
			})
			return
		}

		// Step 3：Nonce 防重放检测
		if len(nonce) != nonceLen {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "INVALID_NONCE",
				"message": "X-Pay-Nonce must be exactly 16 characters",
			})
			return
		}
		nonceKey := "nonce:" + nonce
		ok, err := rdb.SetNX(c.Request.Context(), nonceKey, "1", nonceTTL).Result()
		if err != nil {
			logger.Error("auth: redis nonce check failed", zap.Error(err), zap.String("app_id", appID))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_SERVER_ERROR"})
			return
		}
		if !ok {
			logger.Warn("auth: replay attack detected",
				zap.String("app_id", appID),
				zap.String("nonce", nonce),
				zap.String("ip", c.ClientIP()),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "REPLAY_ATTACK",
				"message": "Nonce already used, please generate a new request",
			})
			return
		}

		// Step 4：读取请求体（限制大小，读后恢复供下游使用）
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBodySize))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": "REQUEST_READ_ERROR"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		// Step 5：查询商户 SecretKey 并校验 HMAC-SHA256 签名
		secretKey, err := secretKeyFn(appID)
		if err != nil {
			logger.Warn("auth: merchant not found", zap.String("app_id", appID))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "INVALID_APP_ID",
				"message": "Merchant not found or disabled",
			})
			return
		}

		// 签名原文：timestamp + "\n" + nonce + "\n" + raw_body
		sigPayload := tsStr + "\n" + nonce + "\n" + string(body)
		mac := hmac.New(sha256.New, []byte(secretKey))
		mac.Write([]byte(sigPayload))
		expectedSig := hex.EncodeToString(mac.Sum(nil))

		// 时序安全比较，防止时序攻击
		if !hmac.Equal([]byte(expectedSig), []byte(signature)) {
			logger.Warn("auth: signature mismatch",
				zap.String("app_id", appID),
				zap.String("ip", c.ClientIP()),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "INVALID_SIGNATURE",
				"message": "Request signature verification failed",
			})
			return
		}

		logger.Info("auth: signature verified", zap.String("app_id", appID))
		c.Set("merchant_app_id", appID)
		c.Next()
	}
}
