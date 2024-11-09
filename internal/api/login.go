package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/imroc/req/v3"
	"golang.org/x/exp/maps"

	"github.com/kkkunny/HuggingChatAPI/internal/config"
)

// Login 登录
func Login(ctx context.Context, username string, password string) ([]*http.Cookie, error) {
	cli := globalClient.Clone()
	cli.SetCommonHeader("origin", config.HuggingChatDomain)

	loginResp, err := login(ctx, cli, &loginRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		return nil, err
	}
	cli.SetCommonCookies(loginResp.Cookies...)

	chatLoginResp, err := chatLogin(ctx, cli)
	if err != nil {
		return nil, err
	}
	cli.SetCommonCookies(chatLoginResp.Cookies...)

	authorizeOauthResp, err := authorizeOauth(ctx, cli, chatLoginResp.Location.String())
	if err != nil {
		return nil, err
	}

	loginCallbackResp, err := loginCallback(ctx, cli, authorizeOauthResp.Location.String())
	if err != nil {
		return nil, err
	}
	cli.SetCommonCookies(loginCallbackResp.Cookies...)

	cookies := make(map[string]*http.Cookie, len(cli.Cookies))
	for _, cookie := range cli.Cookies {
		cookies[cookie.Name] = cookie
	}
	return maps.Values(cookies), nil
}

type loginRequest struct {
	Location string
	Username string
	Password string
}

type loginResponse struct {
	Cookies []*http.Cookie
}

func login(ctx context.Context, cli *req.Client, req *loginRequest) (*loginResponse, error) {
	resp, err := cli.R().
		SetContext(ctx).
		SetContentType("application/x-www-form-urlencoded").
		SetBodyString(fmt.Sprintf("username=%s&password=%s", req.Username, req.Password)).
		Post(fmt.Sprintf("%s/login", config.HuggingChatDomain))
	if err != nil {
		return nil, err
	} else if resp.GetStatusCode() != http.StatusFound {
		return nil, fmt.Errorf("http error: code=%d, status=%s", resp.GetStatusCode(), resp.GetStatus())
	}
	return &loginResponse{Cookies: resp.Cookies()}, nil
}

type chatLoginResponse struct {
	Location *url.URL
	Cookies  []*http.Cookie
}

func chatLogin(ctx context.Context, cli *req.Client) (*chatLoginResponse, error) {
	resp, err := cli.R().
		SetContext(ctx).
		SetContentType("application/x-www-form-urlencoded").
		SetHeaders(map[string]string{
			"accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		}).
		Post(fmt.Sprintf("%s/chat/login", config.HuggingChatDomain))
	if err != nil {
		return nil, err
	} else if resp.GetStatusCode() != http.StatusSeeOther {
		return nil, fmt.Errorf("http error: code=%d, status=%s", resp.GetStatusCode(), resp.GetStatus())
	}
	location, err := resp.Location()
	if err != nil {
		return nil, err
	}
	return &chatLoginResponse{
		Location: location,
		Cookies:  resp.Cookies(),
	}, nil
}

type authorizeOauthResponse struct {
	Location *url.URL
}

func authorizeOauth(ctx context.Context, cli *req.Client, urlStr string) (*authorizeOauthResponse, error) {
	resp, err := cli.R().
		SetContext(ctx).
		SetHeaders(map[string]string{
			"accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		}).
		Get(urlStr)
	if err != nil {
		return nil, err
	} else if resp.GetStatusCode() != http.StatusSeeOther {
		return nil, fmt.Errorf("http error: code=%d, status=%s", resp.GetStatusCode(), resp.GetStatus())
	}
	location, err := resp.Location()
	if err != nil {
		return nil, err
	}
	return &authorizeOauthResponse{Location: location}, nil
}

type loginCallbackResponse struct {
	Cookies []*http.Cookie
}

func loginCallback(ctx context.Context, cli *req.Client, urlStr string) (*loginCallbackResponse, error) {
	resp, err := cli.R().
		SetContext(ctx).
		Get(urlStr)
	if err != nil {
		return nil, err
	} else if resp.GetStatusCode() != http.StatusFound {
		return nil, fmt.Errorf("http error: code=%d, status=%s", resp.GetStatusCode(), resp.GetStatus())
	}
	return &loginCallbackResponse{Cookies: resp.Cookies()}, nil
}
