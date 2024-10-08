package ratelimit

import (
	"net/http"
	"strconv"
	"time"

	"github.com/elisasre/go-common/v2/httputil"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

type (
	KeyFunc func(*gin.Context) (key string, limit *int, err error)
	ErrFunc func(*gin.Context, error) (shouldReturn bool)
)

const (
	HeaderReset     = "X-Ratelimit-Reset"
	HeaderLimit     = "X-Ratelimit-Limit"
	HeaderRemaining = "X-Ratelimit-Remaining"
)

// New creates a distributed rate limiter middleware using redis for state management.
func New(rdb *redis.Client, key KeyFunc, errFunc ErrFunc) gin.HandlerFunc {
	limiter := redis_rate.NewLimiter(rdb)
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		key, limit, err := key(c)
		if err != nil {
			c.JSON(400, httputil.ErrorResponse{Code: 400, Message: err.Error()})
			c.Abort()
			return
		}

		if limit == nil || rdb == nil {
			c.Next()
			return
		}

		res, err := limiter.Allow(ctx, key, redis_rate.PerMinute(*limit))
		if err != nil {
			if shouldReturn := errFunc(c, err); shouldReturn {
				return
			}
		} else {
			reset := time.Now().Add(res.ResetAfter)
			c.Header(HeaderReset, strconv.Itoa(int(reset.Unix())))
			c.Header(HeaderLimit, strconv.Itoa(*limit))
			c.Header(HeaderRemaining, strconv.Itoa(res.Remaining))
			if res.Allowed <= 0 {
				c.JSON(http.StatusTooManyRequests, httputil.ErrorResponse{Code: http.StatusTooManyRequests, Message: "rate limit exceeded"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
