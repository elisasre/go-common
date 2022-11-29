package common

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redis_rate/v9"
)

type (
	KeyFunc func(*gin.Context) (key string, limit *int, err error)
	ErrFunc func(*gin.Context, error) (shouldReturn bool)
)

// RedisRateLimiter ...
func RedisRateLimiter(opts *redis.Options, key KeyFunc, errFunc ErrFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		rdb := redis.NewClient(opts)
		limiter := redis_rate.NewLimiter(rdb)
		key, limit, err := key(c)
		if err != nil {
			c.JSON(400, ErrorResponse{Code: 400, Message: err.Error()})
			c.Abort()
			return
		}
		if limit != nil {
			res, err := limiter.Allow(ctx, key, redis_rate.PerMinute(PtrValue(limit)))
			if err == nil {
				reset := time.Now().Add(res.ResetAfter)
				c.Header("X-Ratelimit-Reset", reset.String())
				c.Header("X-Ratelimit-Limit", strconv.Itoa(PtrValue(limit)))
				c.Header("X-Ratelimit-Remaining", strconv.Itoa(res.Remaining))
				if res.Allowed <= 0 {
					c.JSON(http.StatusTooManyRequests,
						ErrorResponse{Code: http.StatusTooManyRequests, Message: "rate limit exceeded"},
					)
					c.Abort()
					return
				}
			} else {
				shouldReturn := errFunc(c, err)
				if shouldReturn {
					return
				}
			}
		}
		c.Next()
	}
}
