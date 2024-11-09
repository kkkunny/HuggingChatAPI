package config

import (
	"net/http"
	"net/url"
	"os"

	stlerr "github.com/kkkunny/stl/error"
	stlval "github.com/kkkunny/stl/value"
)

var Proxy func(*http.Request) (*url.URL, error)

func init() {
	proxyStr := stlval.Ternary(os.Getenv("https_proxy") != "", os.Getenv("https_proxy"), os.Getenv("HTTPS_PROXY"))
	if proxyStr == "" {
		return
	}
	proxy := stlerr.MustWith(url.Parse(proxyStr))
	Proxy = http.ProxyURL(proxy)
}
