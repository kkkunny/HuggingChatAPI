package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	request "github.com/imroc/req/v3"
	stlerr "github.com/kkkunny/stl/error"
	stlval "github.com/kkkunny/stl/value"

	"github.com/kkkunny/HuggingChatAPI/config"
)

func sendDefaultHttpRequest[Result any](ctx context.Context, method string, reqHandler func(r *request.Request) *request.Request, cookies []*http.Cookie, format string, a ...any) (*Result, error) {
	customRet := !stlval.Is[string](stlval.Default[Result]()) && !stlval.Is[request.Response](stlval.Default[Result]())

	uri, err := stlerr.ErrorWith(url.JoinPath(config.HuggingChatDomain, fmt.Sprintf(format, a...)))
	if err != nil {
		return nil, err
	}

	if reqHandler == nil {
		reqHandler = func(req *request.Request) *request.Request { return req }
	}
	req := reqHandler(globalHttpClient.R().
		SetContext(ctx).
		SetCookies(cookies...).
		SetHeader("origin", config.HuggingChatDomain),
	)
	if customRet {
		req = req.SetSuccessResult(stlval.Default[Result]())
	}
	resp, err := stlerr.ErrorWith(req.Send(method, uri))
	if err != nil {
		return nil, err
	} else if resp.GetStatusCode() != http.StatusOK {
		return nil, stlerr.Errorf("http error: code=%d, status=%s", resp.GetStatusCode(), resp.GetStatus())
	}

	switch any(stlval.Default[Result]()).(type) {
	case request.Response:
		return any(resp).(*Result), nil
	case string:
		return stlval.Ptr(any(resp.String()).(Result)), nil
	default:
		if resp.ResultState() != request.SuccessState {
			return nil, stlerr.Errorf("parse http result error: code=%d, status=%s", resp.GetStatusCode(), resp.GetStatus())
		}
		res, ok := resp.SuccessResult().(*Result)
		if !ok {
			return nil, stlerr.Errorf("parse http result error: code=%d, status=%s", resp.GetStatusCode(), resp.GetStatus())
		}
		return res, nil
	}
}
