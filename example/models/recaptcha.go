package models

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/MixinNetwork/ocean.one/example/config"
	"github.com/MixinNetwork/ocean.one/example/session"
)

type recaptchaResp struct {
	Success bool `json:"success"`
}

var httpClient *http.Client

func verifyRecaptcha(ctx context.Context, response string) (bool, error) {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	remoteIP := session.RemoteAddress(ctx)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s?secret=%s&response=%s&remoteip=%s", config.RecaptchaURL, config.RecaptchaSecret, response, remoteIP), nil)
	if err != nil {
		return false, session.BadDataError(ctx)
	}
	req.Close = true
	resp, err := httpClient.Do(req)
	if err != nil {
		return false, session.BadDataError(ctx)
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, session.BadDataError(ctx)
	}
	var captcha recaptchaResp
	err = json.Unmarshal(bytes, &captcha)
	if err != nil {
		return false, session.BadDataError(ctx)
	}
	return captcha.Success, nil
}
