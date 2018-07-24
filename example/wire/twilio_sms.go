package wire

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/MixinNetwork/ocean.one/example/config"
)

func TwilioSendVerificationCode(phone, code string) error {
	body := fmt.Sprintf("Ocean code %s", code)
	callback := "https://example.ocean.one/callbacks/twilio/" + phone[1:] + "/" + code
	return TwilioSendSMS(phone, body, callback)
}

func TwilioSendSMS(phone, body, callback string) error {
	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", config.TwilioAccountSid)
	params := url.Values{
		"To": []string{phone},
		"MessagingServiceSid": []string{config.TwilioMessagingServiceSid},
		"Body":                []string{body},
	}
	if strings.HasPrefix(callback, "https://") {
		params["StatusCallback"] = []string{callback}
	}

	client := &http.Client{}
	req, _ := http.NewRequest("POST", endpoint, strings.NewReader(params.Encode()))
	req.SetBasicAuth(config.TwilioAccountSid, config.TwilioAuthToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
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
