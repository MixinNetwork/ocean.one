package models

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/MixinNetwork/ocean.one/example/config"
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

type kickboxRequest struct {
	Result string `json:"result"`
}

func ValidateEmailFormat(ctx context.Context, email string) (string, error) {
	email = strings.TrimSpace(email)
	var emailRegexp = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	if !emailRegexp.MatchString(email) {
		return "", session.EmailInvalidFormatError(ctx, email)
	}

	client := &http.Client{Timeout: config.ExternalNetworkTimeout}
	resp, err := client.Get(fmt.Sprintf("https://api.kickbox.io/v2/verify?email=%s&apikey=%s", email, config.KickboxApikey))
	if err != nil {
		return "", session.EmailInvalidFormatError(ctx, email)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return email, nil
	}
	var result kickboxRequest
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", session.EmailInvalidFormatError(ctx, email)
	}
	if result.Result == "deliverable" {
		return email, nil
	}
	return "", session.EmailInvalidFormatError(ctx, email)
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
