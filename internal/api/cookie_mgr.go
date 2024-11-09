package api

import (
	"context"
	"errors"
	"net/http"

	stlslices "github.com/kkkunny/stl/container/slices"
)

type cookieMgr interface {
	Cookies(ctx context.Context) ([]*http.Cookie, error)
	Refresh(ctx context.Context) ([]*http.Cookie, error)
}

type tokenCookieMgr struct {
	token string
}

func newTokenCookieMgr(token string) *tokenCookieMgr {
	return &tokenCookieMgr{token: token}
}

func (mgr *tokenCookieMgr) Cookies(_ context.Context) ([]*http.Cookie, error) {
	return []*http.Cookie{{Name: "hf-chat", Value: mgr.token}}, nil
}

func (mgr *tokenCookieMgr) Refresh(_ context.Context) ([]*http.Cookie, error) {
	return nil, errors.New("can not refresh token cookie")
}

type accountCookieMgr struct {
	username string
	password string
}

func newAccountCookieMgr(usr, pwd string) *accountCookieMgr {
	return &accountCookieMgr{
		username: usr,
		password: pwd,
	}
}

func (mgr *accountCookieMgr) Cookies(ctx context.Context) ([]*http.Cookie, error) {
	cookies := globalCookieCache.Get(mgr.username)
	if len(stlslices.Filter(cookies, func(_ int, cookie *http.Cookie) bool {
		return cookie.Name == "hf-chat"
	})) == 0 {
		return mgr.Refresh(ctx)
	}
	return globalCookieCache.Get(mgr.username), nil
}

func (mgr *accountCookieMgr) Refresh(ctx context.Context) ([]*http.Cookie, error) {
	cookies, err := Login(ctx, mgr.username, mgr.password)
	if err != nil {
		return nil, err
	}
	err = globalCookieCache.Set(mgr.username, cookies)
	return cookies, err
}
