package wire

import (
	"fmt"

	"github.com/MixinNetwork/ocean.one/example/config"
	"gopkg.in/gomail.v2"
)

const (
	SESProviderAWS = "amazon"
)

func SendVerificationCodeByEmail(email, code string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", m.FormatAddress(config.SMTPFromEmail, config.SMTPFromName))
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Ocean ONE verification code")
	m.SetBody("text/html", fmt.Sprintf(verificationCodeEmailTemplate, email, code))

	return sendMail(m)
}

const verificationCodeEmailTemplate = `<p>Hello %s!</p>
<p>Ocean code %s</p>`
