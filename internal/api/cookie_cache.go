package api

import (
	"encoding/json"
	"errors"
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
	data, err := stlerr.ErrorWith(os.ReadFile(config.CookieCachePath))
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	err = stlerr.ErrorWrap(json.Unmarshal(data, &cache.data))
	return err
}

func (cache *cookieCache) Save() error {
	err := stlerr.ErrorWrap(os.MkdirAll(filepath.Dir(config.CookieCachePath), 0750))
	if err != nil {
		return err
	}
	file, err := stlerr.ErrorWith(os.OpenFile(config.CookieCachePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666))
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := stlerr.ErrorWith(json.MarshalIndent(cache.data, "", "  "))
	if err != nil {
		return err
	}
	_, err = stlerr.ErrorWith(file.Write(data))
	return err
}

func (cache *cookieCache) Get(usr string) []*http.Cookie {
	return cache.data[usr]
}

func (cache *cookieCache) Set(usr string, cookies []*http.Cookie) error {
	cache.data[usr] = cookies
	return cache.Save()
}
