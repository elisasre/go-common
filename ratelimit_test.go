package common

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func ExampleRedisRateLimiter() {
	// allow 2 requests per user per minute, in case of error (like connectivity issue to redis) skip limiter
	_ = RedisRateLimiter(
		&redis.Options{Addr: "localhost:6379"},
		func(c *gin.Context) (key string, limit *int, err error) {
			return "username", Int(2), nil
		},
		func(c *gin.Context, err error) bool {
			log.Printf("%+v\n", err)
			return false
		},
	)
	// then apply to single route or group
	// r.GET("/ratelimit", MustAuth(opts), redisLimiter, opts.ratelimit)
}
