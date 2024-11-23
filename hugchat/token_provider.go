package hugchat

import (
	"context"
	"fmt"
	"net/http"

	stlslices "github.com/kkkunny/stl/container/slices"
	stlerr "github.com/kkkunny/stl/error"

	"github.com/kkkunny/HuggingChatAPI/internal/api"
)

const tokenCookieKey = "hf-chat"

var RefreshTokenError = fmt.Errorf("refresh token error")

type TokenProvider interface {
	RefreshToken(ctx context.Context) ([]*http.Cookie, error)
	GetToken(ctx context.Context) ([]*http.Cookie, error)
}

type directTokenProvider struct {
	token string
}

func NewDirectTokenProvider(token string) TokenProvider {
	return &directTokenProvider{token: token}
}

func (p *directTokenProvider) RefreshToken(_ context.Context) ([]*http.Cookie, error) {
	return nil, stlerr.ErrorWrap(RefreshTokenError)
}

func (p *directTokenProvider) GetToken(_ context.Context) ([]*http.Cookie, error) {
	return []*http.Cookie{{Name: tokenCookieKey, Value: p.token}}, nil
}

type accountTokenProvider struct {
	username string
	password string
}

func NewAccountTokenProvider(usr, pwd string) TokenProvider {
	return &accountTokenProvider{username: usr, password: pwd}
}

func (p *accountTokenProvider) RefreshToken(ctx context.Context) ([]*http.Cookie, error) {
	token, err := api.Login(ctx, p.username, p.password)
	if err != nil {
		return nil, err
	}
	err = globalCookieCache.Set(p.username, token)
	return token, err
}

func (p *accountTokenProvider) GetToken(ctx context.Context) ([]*http.Cookie, error) {
	cookies := globalCookieCache.Get(p.username)
	if len(stlslices.Filter(cookies, func(_ int, cookie *http.Cookie) bool {
		return cookie.Name == tokenCookieKey
	})) == 0 {
		return p.RefreshToken(ctx)
	}
	return globalCookieCache.Get(p.username), nil
}
