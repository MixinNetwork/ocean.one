package mixin

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
)

var httpClient *http.Client

func getMixinUrl(uri string) string {
	return "https://api.mixin.one" + uri
}

func (client *Client) SendRequest(ctx context.Context, method, uri string, payload []byte) ([]byte, error) {
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	accessToken, err := client.signAuthenticationToken(method, uri, string(payload))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, getMixinUrl(uri), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Close = true
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
