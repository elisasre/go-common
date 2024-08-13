package ratelimit

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

type (
	KeyFunc func(*gin.Context) (key string, limit *int, err error)
	ErrFunc func(*gin.Context, error) (shouldReturn bool)
)

const (
	RatelimitReset     = "X-Ratelimit-Reset"
	RatelimitLimit     = "X-Ratelimit-Limit"
	RatelimitRemaining = "X-Ratelimit-Remaining"
)

func (e ErrorResponse) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

// ErrorResponse provides HTTP error response.
type ErrorResponse struct {
	Code      uint   `json:"code,omitempty" example:"400"`
	Message   string `json:"message" example:"Bad request"`
	ErrorType string `json:"error_type,omitempty" example:"invalid_scope"`
}

// RedisRateLimiter ...
func RedisRateLimiter(rdb *redis.Client, key KeyFunc, errFunc ErrFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		limiter := redis_rate.NewLimiter(rdb)
		key, limit, err := key(c)
		if err != nil {
			c.JSON(400, ErrorResponse{Code: 400, Message: err.Error()})
			c.Abort()
			return
		}
		if limit == nil || rdb == nil {
			c.Next()
			return
		}
		res, err := limiter.Allow(ctx, key, redis_rate.PerMinute(*limit))
		if err == nil {
			reset := time.Now().Add(res.ResetAfter)
			c.Header(RatelimitReset, strconv.Itoa(int(reset.Unix())))
			c.Header(RatelimitLimit, strconv.Itoa(*limit))
			c.Header(RatelimitRemaining, strconv.Itoa(res.Remaining))
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
		c.Next()
	}
}
