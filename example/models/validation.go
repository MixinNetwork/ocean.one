package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/nyaruka/phonenumbers"
	"golang.org/x/crypto/bcrypt"
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

func ValidateAndEncryptPassword(ctx context.Context, password string) (string, error) {
	password = strings.TrimSpace(password)
	if len(password) < 8 {
		return password, session.PasswordTooSimpleError(ctx)
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return password, session.ServerError(ctx, err)
	}
	return string(hashedPassword), nil
}
