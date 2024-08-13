package ratelimit_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/elisasre/go-common/v2/middleware/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

const testUser = "test-user"

func setupRouter(mw gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.GET("/healthz", mw, func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	return r
}

func TestRedisRateLimiterAlways(t *testing.T) {
	s, err := miniredis.Run()
	require.Equal(t, err, nil)
	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	alwaysRateLimiter := ratelimit.RedisRateLimiter(redisClient,
		func(c *gin.Context) (key string, limit *int, err error) {
			return testUser, intPtr(2), nil
		},
		func(c *gin.Context, err error) bool {
			if err != nil {
				t.Log(err)
			}
			return false
		},
	)

	router := setupRouter(alwaysRateLimiter)
	require.Equal(t, err, nil)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/healthz", nil)
	require.Equal(t, err, nil)
	router.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	require.Equal(t, "ok", w.Body.String())
	require.Equal(t, "2", w.Result().Header.Get(ratelimit.RatelimitLimit))
	require.Equal(t, "1", w.Result().Header.Get(ratelimit.RatelimitRemaining))

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req)
	require.Equal(t, 200, w2.Code)
	require.Equal(t, "ok", w2.Body.String())
	require.Equal(t, "2", w2.Result().Header.Get(ratelimit.RatelimitLimit))
	require.Equal(t, "0", w2.Result().Header.Get(ratelimit.RatelimitRemaining))

	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req)
	require.Equal(t, 429, w3.Code)
	require.Equal(t, `{"code":429,"message":"rate limit exceeded"}`, w3.Body.String())
	require.Equal(t, "2", w3.Result().Header.Get(ratelimit.RatelimitLimit))
	require.Equal(t, "0", w3.Result().Header.Get(ratelimit.RatelimitRemaining))
}

func TestRedisRateLimiterSkip(t *testing.T) {
	s, err := miniredis.Run()
	require.Equal(t, err, nil)
	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	skipRateLimiter := ratelimit.RedisRateLimiter(redisClient,
		func(c *gin.Context) (key string, limit *int, err error) {
			return "", nil, nil
		},
		func(c *gin.Context, err error) bool {
			if err != nil {
				t.Log(err)
			}
			return false
		},
	)

	router := setupRouter(skipRateLimiter)
	require.Equal(t, err, nil)
	for i := 1; i < 5; i++ {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/healthz", nil)
		require.Equal(t, err, nil)
		router.ServeHTTP(w, req)
		require.Equal(t, 200, w.Code)
		require.Equal(t, "ok", w.Body.String())

		require.Equal(t, "", w.Result().Header.Get(ratelimit.RatelimitLimit))
		require.Equal(t, "", w.Result().Header.Get(ratelimit.RatelimitRemaining))
	}
}

func TestRedisRateLimiterForce(t *testing.T) {
	s, err := miniredis.Run()
	require.Equal(t, err, nil)
	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	forceRateLimiter := ratelimit.RedisRateLimiter(redisClient,
		func(c *gin.Context) (key string, limit *int, err error) {
			return testUser, intPtr(2), nil
		},
		func(c *gin.Context, err error) bool {
			if err != nil {
				t.Log(err)
			}
			c.JSON(http.StatusBadRequest,
				ratelimit.ErrorResponse{Code: http.StatusBadRequest, Message: err.Error()},
			)
			c.Abort()
			return true
		},
	)
	router := setupRouter(forceRateLimiter)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/healthz", nil)
	require.Equal(t, err, nil)
	router.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	require.Equal(t, "ok", w.Body.String())
	require.Equal(t, "2", w.Result().Header.Get(ratelimit.RatelimitLimit))
	require.Equal(t, "1", w.Result().Header.Get(ratelimit.RatelimitRemaining))

	s.SetError("server is unavailable")

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req)
	require.Equal(t, 400, w2.Code)
	require.Equal(t, `{"code":400,"message":"server is unavailable"}`, w2.Body.String())
	require.Equal(t, "", w2.Result().Header.Get(ratelimit.RatelimitLimit))
	require.Equal(t, "", w2.Result().Header.Get(ratelimit.RatelimitRemaining))
}

func TestRedisRateLimiterNil(t *testing.T) {
	nilLimiter := ratelimit.RedisRateLimiter(nil,
		func(c *gin.Context) (key string, limit *int, err error) {
			return testUser, intPtr(2), nil
		},
		func(c *gin.Context, err error) bool {
			if err != nil {
				t.Log(err)
			}
			c.JSON(http.StatusBadRequest,
				ratelimit.ErrorResponse{Code: http.StatusBadRequest, Message: err.Error()},
			)
			c.Abort()
			return true
		},
	)
	router := setupRouter(nilLimiter)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/healthz", nil)
	require.Equal(t, err, nil)
	router.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	require.Equal(t, "ok", w.Body.String())
	require.Equal(t, "", w.Result().Header.Get(ratelimit.RatelimitLimit))
	require.Equal(t, "", w.Result().Header.Get(ratelimit.RatelimitRemaining))
}

func intPtr(i int) *int { return &i }
