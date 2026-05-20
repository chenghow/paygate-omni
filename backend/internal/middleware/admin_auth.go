// Package middleware 包含 Admin 会话令牌校验中间件。
package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// AdminAuth 验证请求头 Authorization: Bearer <token> 对应的 Redis 会话是否有效。
// 每次成功验证后自动续期 24 小时。
func AdminAuth(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"code": "UNAUTHORIZED", "message": "missing bearer token"})
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"code": "UNAUTHORIZED", "message": "empty token"})
			return
		}

		key := "admin_session:" + token
		val, err := rdb.Get(context.Background(), key).Result()
		if err != nil || val != "ok" {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"code": "UNAUTHORIZED", "message": "invalid or expired token"})
			return
		}

		// 活跃使用时自动续期
		rdb.Expire(context.Background(), key, 24*time.Hour)
		c.Next()
	}
}
