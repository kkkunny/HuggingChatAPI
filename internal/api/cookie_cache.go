package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	stlerr "github.com/kkkunny/stl/error"

	"github.com/kkkunny/HuggingChatAPI/internal/config"
)

var globalCookieCache *cookieCache

func init() {
	globalCookieCache = newCookieCache()
	stlerr.Must(globalCookieCache.Load())
}

type cookieCache struct {
	data map[string][]*http.Cookie
}

func newCookieCache() *cookieCache {
	return &cookieCache{data: make(map[string][]*http.Cookie)}
}

func (cache *cookieCache) Load() error {
	data, err := os.ReadFile(config.CookieCachePath)
	if err != nil && os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	err = json.Unmarshal(data, &cache.data)
	return err
}

func (cache *cookieCache) Save() error {
	err := os.MkdirAll(filepath.Dir(config.CookieCachePath), 0750)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(config.CookieCachePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.MarshalIndent(cache.data, "", "  ")
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	return err
}

func (cache *cookieCache) Get(usr string) []*http.Cookie {
	return cache.data[usr]
}

func (cache *cookieCache) Set(usr string, cookies []*http.Cookie) error {
	cache.data[usr] = cookies
	return cache.Save()
}
