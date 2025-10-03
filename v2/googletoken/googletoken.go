package googletoken

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/sts/v1"
)

type k8sSaTokenSource struct {
	client      STSExchanger
	saPath      string
	stsAudience string
}

type STSClient struct {
	service *sts.V1Service
}

type STSExchanger interface {
	ExchangeToken(req *sts.GoogleIdentityStsV1ExchangeTokenRequest) (*sts.GoogleIdentityStsV1ExchangeTokenResponse, error)
}

func NewSTSClient(ctx context.Context) (*STSClient, error) {
	service, err := sts.NewService(ctx, option.WithoutAuthentication())
	if err != nil {
		return nil, fmt.Errorf("failed to create STS service: %w", err)
	}
	return &STSClient{service: service.V1}, nil
}

func (c *STSClient) ExchangeToken(req *sts.GoogleIdentityStsV1ExchangeTokenRequest) (*sts.GoogleIdentityStsV1ExchangeTokenResponse, error) {
	return c.service.Token(req).Do()
}

func NewTokenSourceFromSAToken(client STSExchanger, saPath, stsAudience string) (oauth2.TokenSource, error) {
	file, err := os.Open(saPath)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			return nil, fmt.Errorf("token file does not exist")
		case os.IsPermission(err):
			return nil, fmt.Errorf("permission to token filedenied")
		default:
			return nil, fmt.Errorf("cannot access token file: %w", err)
		}
	}
	defer file.Close()

	ts := &k8sSaTokenSource{
		client:      client,
		saPath:      saPath,
		stsAudience: stsAudience,
	}

	return ts, nil
}

func (k k8sSaTokenSource) Token() (*oauth2.Token, error) {
	tokenBytes, err := os.ReadFile(k.saPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read service account file: %w", err)
	}

	res, err := k.client.ExchangeToken(&sts.GoogleIdentityStsV1ExchangeTokenRequest{
		Audience:           k.stsAudience,
		GrantType:          "urn:ietf:params:oauth:grant-type:token-exchange",
		RequestedTokenType: "urn:ietf:params:oauth:token-type:access_token",
		Scope:              "https://www.googleapis.com/auth/cloud-platform",
		SubjectToken:       string(tokenBytes),
		SubjectTokenType:   "urn:ietf:params:oauth:token-type:jwt",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}

	slog.Debug("refreshing token", slog.Int64("expires_in", res.ExpiresIn))
	return &oauth2.Token{
		AccessToken: res.AccessToken,
		ExpiresIn:   res.ExpiresIn,
		Expiry:      time.Now().Add(time.Duration(res.ExpiresIn) * time.Second),
	}, nil
}
