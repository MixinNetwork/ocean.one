package wire

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MixinNetwork/ocean.one/example/config"
	"github.com/MixinNetwork/ocean.one/example/uuid"
)

func TelesignSendVerificationCode(phone, code string) error {
	method := "POST"
	authMethod := "HMAC-SHA256"
	contentType := "application/x-www-form-urlencoded"
	t := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	nonce := uuid.NewV4().String()
	q := url.Values{}
	q.Add("message", "")
	q.Add("message_type", "OTP")
	q.Add("phone_number", phone)
	q.Add("template", "Ocean code $$CODE$$ ")
	q.Add("verify_code", code)
	query := q.Encode()

	req, err := http.NewRequest(method, "https://rest-ww.telesign.com/v1/verify/sms", strings.NewReader(query))
	if err != nil {
		return err
	}
	key, err := base64.StdEncoding.DecodeString(config.TelesignSecret)
	if err != nil {
		return err
	}
	stringToSign := fmt.Sprintf("%s\n%s\n%s\nx-ts-auth-method:%s\nx-ts-nonce:%s\n%s\n%s", method, contentType, t, authMethod, nonce, query, "/v1/verify/sms")
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(stringToSign))
	signature := strings.TrimSpace(base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	authorization := fmt.Sprintf("TSA %s:%s", config.TelesignId, signature)
	req.Header.Add("Authorization", authorization)
	req.Header.Add("Date", t)
	req.Header.Add("x-ts-auth-method", authMethod)
	req.Header.Add("Content-Type", contentType)
	req.Header.Add("x-ts-nonce", nonce)
	req.Header.Add("User-Agent", "TeleSignSDK/golang-1.9 mixin net/http/persistent")
	client := &http.Client{Timeout: config.ExternalNetworkTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	} else {
		return fmt.Errorf("SMS gateway error code: %d", resp.StatusCode)
	}
}
