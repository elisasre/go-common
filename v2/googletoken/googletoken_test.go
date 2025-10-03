package googletoken

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/sts/v1"
)

type mockSTSExchanger struct {
	exchangeTokenFunc func(req *sts.GoogleIdentityStsV1ExchangeTokenRequest) (*sts.GoogleIdentityStsV1ExchangeTokenResponse, error)
}

func (m *mockSTSExchanger) ExchangeToken(req *sts.GoogleIdentityStsV1ExchangeTokenRequest) (*sts.GoogleIdentityStsV1ExchangeTokenResponse, error) {
	if m.exchangeTokenFunc != nil {
		return m.exchangeTokenFunc(req)
	}
	return &sts.GoogleIdentityStsV1ExchangeTokenResponse{
		AccessToken: "test-access-token",
		ExpiresIn:   3600,
		TokenType:   "Bearer",
	}, nil
}

func TestNewTokenSourceFromSAToken(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (string, func())
		client      STSExchanger
		stsAudience string
		wantErr     bool
		errContains string
	}{
		{
			name: "success with valid token file",
			setup: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "sa-token-*")
				require.NoError(t, err)
				_, err = tmpFile.Write([]byte("test-token"))
				require.NoError(t, err)
				tmpFile.Close()
				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
			client:      &mockSTSExchanger{},
			stsAudience: "test-audience",
			wantErr:     false,
		},
		{
			name: "error when file does not exist",
			setup: func() (string, func()) {
				return "/non/existent/file", func() {}
			},
			client:      &mockSTSExchanger{},
			stsAudience: "test-audience",
			wantErr:     true,
			errContains: "token file does not exist",
		},
		{
			name: "error when permission denied",
			setup: func() (string, func()) {
				tmpDir, err := os.MkdirTemp("", "sa-token-test")
				require.NoError(t, err)
				tmpFile := filepath.Join(tmpDir, "token")
				err = os.WriteFile(tmpFile, []byte("test-token"), 0o000)
				require.NoError(t, err)
				return tmpFile, func() { os.RemoveAll(tmpDir) }
			},
			client:      &mockSTSExchanger{},
			stsAudience: "test-audience",
			wantErr:     true,
			errContains: "permission to token filedenied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenPath, cleanup := tt.setup()
			defer cleanup()

			ts, err := NewTokenSourceFromSAToken(tt.client, tokenPath, tt.stsAudience)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, ts)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ts)
				tokenSource, ok := ts.(*k8sSaTokenSource)
				assert.True(t, ok)
				assert.Equal(t, tokenPath, tokenSource.saPath)
				assert.Equal(t, tt.stsAudience, tokenSource.stsAudience)
				assert.Equal(t, tt.client, tokenSource.client)
			}
		})
	}
}

func TestK8sSaTokenSource_Token(t *testing.T) {
	tests := []struct {
		name              string
		setup             func() (string, func())
		mockExchangerFunc func(req *sts.GoogleIdentityStsV1ExchangeTokenRequest) (*sts.GoogleIdentityStsV1ExchangeTokenResponse, error)
		stsAudience       string
		wantErr           bool
		errContains       string
		validateToken     func(t *testing.T, token *sts.GoogleIdentityStsV1ExchangeTokenRequest)
	}{
		{
			name: "successful token exchange",
			setup: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "sa-token-*")
				require.NoError(t, err)
				_, err = tmpFile.Write([]byte("test-jwt-token"))
				require.NoError(t, err)
				tmpFile.Close()
				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
			mockExchangerFunc: func(req *sts.GoogleIdentityStsV1ExchangeTokenRequest) (*sts.GoogleIdentityStsV1ExchangeTokenResponse, error) {
				return &sts.GoogleIdentityStsV1ExchangeTokenResponse{
					AccessToken: "exchanged-access-token",
					ExpiresIn:   7200,
					TokenType:   "Bearer",
				}, nil
			},
			stsAudience: "https://example.com/audience",
			wantErr:     false,
			validateToken: func(t *testing.T, req *sts.GoogleIdentityStsV1ExchangeTokenRequest) {
				assert.Equal(t, "https://example.com/audience", req.Audience)
				assert.Equal(t, "urn:ietf:params:oauth:grant-type:token-exchange", req.GrantType)
				assert.Equal(t, "urn:ietf:params:oauth:token-type:access_token", req.RequestedTokenType)
				assert.Equal(t, "https://www.googleapis.com/auth/cloud-platform", req.Scope)
				assert.Equal(t, "test-jwt-token", req.SubjectToken)
				assert.Equal(t, "urn:ietf:params:oauth:token-type:jwt", req.SubjectTokenType)
			},
		},
		{
			name: "error reading service account file",
			setup: func() (string, func()) {
				return "/non/existent/file", func() {}
			},
			mockExchangerFunc: nil,
			stsAudience:       "test-audience",
			wantErr:           true,
			errContains:       "failed to read service account file",
		},
		{
			name: "error exchanging token",
			setup: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "sa-token-*")
				require.NoError(t, err)
				_, err = tmpFile.Write([]byte("test-jwt-token"))
				require.NoError(t, err)
				tmpFile.Close()
				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
			mockExchangerFunc: func(req *sts.GoogleIdentityStsV1ExchangeTokenRequest) (*sts.GoogleIdentityStsV1ExchangeTokenResponse, error) {
				return nil, errors.New("exchange failed: invalid token")
			},
			stsAudience: "test-audience",
			wantErr:     true,
			errContains: "failed to exchange token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenPath, cleanup := tt.setup()
			defer cleanup()

			var capturedRequest *sts.GoogleIdentityStsV1ExchangeTokenRequest
			mockClient := &mockSTSExchanger{
				exchangeTokenFunc: func(req *sts.GoogleIdentityStsV1ExchangeTokenRequest) (*sts.GoogleIdentityStsV1ExchangeTokenResponse, error) {
					capturedRequest = req
					if tt.mockExchangerFunc != nil {
						return tt.mockExchangerFunc(req)
					}
					return nil, errors.New("no mock function provided")
				},
			}

			ts := &k8sSaTokenSource{
				client:      mockClient,
				saPath:      tokenPath,
				stsAudience: tt.stsAudience,
			}

			beforeTime := time.Now()
			token, err := ts.Token()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, token)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, token)

				assert.Equal(t, "exchanged-access-token", token.AccessToken)
				assert.Equal(t, int64(7200), token.ExpiresIn)

				assert.True(t, token.Expiry.After(beforeTime))
				assert.True(t, token.Expiry.Before(time.Now().Add(7201*time.Second)))

				if tt.validateToken != nil && capturedRequest != nil {
					tt.validateToken(t, capturedRequest)
				}
			}
		})
	}
}

func TestK8sSaTokenSource_TokenRefresh(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "sa-token-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte("initial-jwt-token"))
	require.NoError(t, err)
	tmpFile.Close()

	callCount := 0
	mockClient := &mockSTSExchanger{
		exchangeTokenFunc: func(req *sts.GoogleIdentityStsV1ExchangeTokenRequest) (*sts.GoogleIdentityStsV1ExchangeTokenResponse, error) {
			callCount++
			return &sts.GoogleIdentityStsV1ExchangeTokenResponse{
				AccessToken: fmt.Sprintf("token-%d", callCount),
				ExpiresIn:   3600,
				TokenType:   "Bearer",
			}, nil
		},
	}

	ts := &k8sSaTokenSource{
		client:      mockClient,
		saPath:      tmpFile.Name(),
		stsAudience: "test-audience",
	}

	token1, err := ts.Token()
	assert.NoError(t, err)
	assert.Equal(t, "token-1", token1.AccessToken)

	err = os.WriteFile(tmpFile.Name(), []byte("updated-jwt-token"), 0o600)
	require.NoError(t, err)

	token2, err := ts.Token()
	assert.NoError(t, err)
	assert.Equal(t, "token-2", token2.AccessToken)

	assert.Equal(t, 2, callCount)
}

func TestK8sSaTokenSource_ConcurrentAccess(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "sa-token-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte("concurrent-test-token"))
	require.NoError(t, err)
	tmpFile.Close()

	var callCount atomic.Int32
	mockClient := &mockSTSExchanger{
		exchangeTokenFunc: func(req *sts.GoogleIdentityStsV1ExchangeTokenRequest) (*sts.GoogleIdentityStsV1ExchangeTokenResponse, error) {
			callCount.Add(1)
			time.Sleep(10 * time.Millisecond)
			return &sts.GoogleIdentityStsV1ExchangeTokenResponse{
				AccessToken: "concurrent-token",
				ExpiresIn:   3600,
				TokenType:   "Bearer",
			}, nil
		},
	}

	ts := &k8sSaTokenSource{
		client:      mockClient,
		saPath:      tmpFile.Name(),
		stsAudience: "test-audience",
	}

	numGoroutines := 10
	errors := make(chan error, numGoroutines)
	tokens := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			token, err := ts.Token()
			if err != nil {
				errors <- err
			} else {
				tokens <- token.AccessToken
			}
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-errors:
			t.Fatalf("unexpected error in concurrent access: %v", err)
		case token := <-tokens:
			assert.Equal(t, "concurrent-token", token)
		}
	}

	assert.Equal(t, int32(numGoroutines), callCount.Load())
}
