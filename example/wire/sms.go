package wire

import "github.com/MixinNetwork/ocean.one/example/config"

const (
	SMSProviderTelesign = "telesign"
	SMSProviderTwilio   = "twilio"
)

func SendVerificationCode(provider, phone, code string) error {
	if !config.SMSDeliveryEnabled {
		return nil
	}
	switch provider {
	case SMSProviderTwilio:
		if err := TwilioSendVerificationCode(phone, code); err != nil {
			return TelesignSendVerificationCode(phone, code)
		}
	default:
		if err := TelesignSendVerificationCode(phone, code); err != nil {
			return TwilioSendVerificationCode(phone, code)
		}
	}
	return nil
}
