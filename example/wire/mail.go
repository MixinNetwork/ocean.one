package wire

import (
	"github.com/MixinNetwork/ocean.one/example/config"
	"gopkg.in/gomail.v2"
)

func sendMail(m *gomail.Message) error {
	d := gomail.NewPlainDialer(config.SESServer, config.SESPort, config.SESAddress, config.SESPassword)

	return d.DialAndSend(m)
}
