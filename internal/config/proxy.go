package config

import (
	"net/http"
	"net/url"
	"os"
)

var Proxy = func() func(*http.Request) (*url.URL, error) {
	proxy := os.Getenv("https_proxy")
	if proxy == "" {
		proxy = os.Getenv("HTTPS_PROXY")
		if proxy == "" {
			return nil
		}
	}
	proxyUrl, err := url.Parse(proxy)
	if err != nil {
		panic(err)
	}
	return http.ProxyURL(proxyUrl)
}()
