package wire

import (
	"context"

	"github.com/MixinNetwork/ocean.one/example/session"
)

func SendVerificationCode(ctx context.Context, category, provider, receiver, code string) error {
	if category == "PHONE" {
		if err := SendVerificationCodeByPhone(provider, receiver, code); err != nil {
			return session.PhoneSMSDeliveryError(ctx, receiver, err)
		}
	}
	if category == "EMAIL" {
		if err := SendVerificationCodeByEmail(receiver, code); err != nil {
			return session.EmailSMSDeliveryError(ctx, receiver, err)
		}
	}
	return nil
}
