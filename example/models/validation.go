package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/nyaruka/phonenumbers"
)

func ValidatePhoneNumberFormat(ctx context.Context, phone string) (string, error) {
	phone = strings.ToUpper(strings.TrimSpace(phone))
	if !strings.HasPrefix(phone, "+") {
		return "", session.PhoneInvalidFormatError(ctx, phone)
	}

	number, err := phonenumbers.Parse(phone, "US")
	if err != nil {
		return "", session.PhoneInvalidFormatError(ctx, phone)
	}

	if !phonenumbers.IsValidNumber(number) {
		return "", session.PhoneInvalidFormatError(ctx, phone)
	}
	return fmt.Sprintf("+%d%d", number.GetCountryCode(), number.GetNationalNumber()), nil
}
