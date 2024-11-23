package hugchat

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	stlerr "github.com/kkkunny/stl/error"
)

const cookieCachePath = "config/cookies.json"

var globalCookieCache *cookieCache

func init() {
	globalCookieCache = newCookieCache()
	stlerr.Must(globalCookieCache.load())
}

type cookieCache struct {
	lock sync.RWMutex
	data map[string][]*http.Cookie
}

func newCookieCache() *cookieCache {
	return &cookieCache{data: make(map[string][]*http.Cookie)}
}

func (cache *cookieCache) load() error {
	data, err := stlerr.ErrorWith(os.ReadFile(cookieCachePath))
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	err = stlerr.ErrorWrap(json.Unmarshal(data, &cache.data))
	return err
}

func (cache *cookieCache) save() error {
	err := stlerr.ErrorWrap(os.MkdirAll(filepath.Dir(cookieCachePath), 0750))
	if err != nil {
		return err
	}
	file, err := stlerr.ErrorWith(os.OpenFile(cookieCachePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666))
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
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	return cache.data[usr]
}

func (cache *cookieCache) Set(usr string, cookies []*http.Cookie) error {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	cache.data[usr] = cookies
	return cache.save()
}
