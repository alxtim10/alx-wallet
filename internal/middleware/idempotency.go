package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func Idempotency(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}

		val, _ := rdb.Get(context.Background(), key).Result()
		if val != "" {
			c.Data(200, "application/json", []byte(val))
			c.Abort()
			return
		}

		c.Next()

		rdb.Set(context.Background(), key,
			c.Writer.Header().Get("X-Response-Body"), 24*time.Hour)
	}
}
