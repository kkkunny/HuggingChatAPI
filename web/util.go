package main

import (
	"encoding/base64"
	"regexp"

	stlerr "github.com/kkkunny/stl/error"

	"github.com/kkkunny/HuggingChatAPI/hugchat"
)

func parseAuthorization(token string) (hugchat.TokenProvider, error) {
	account, err := base64.StdEncoding.DecodeString(token)
	if err == nil {
		res := regexp.MustCompile(`username=(.+?)&password=(.+)`).FindStringSubmatch(string(account))
		if len(res) != 3 {
			return nil, stlerr.Errorf("invalid token")
		}
		return hugchat.NewAccountTokenProvider(res[1], res[2]), nil
	}
	return hugchat.NewDirectTokenProvider(token), nil
}
