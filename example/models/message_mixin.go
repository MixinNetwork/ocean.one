package models

import (
	"context"
	"encoding/base64"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/ocean.one/example/config"
)

func notifyMesseger(ctx context.Context, recipientId, content string) {
	conversationId := bot.UniqueConversationId(config.ClientId, recipientId)
	data := base64.StdEncoding.EncodeToString([]byte(content))
	bot.PostMessage(ctx, conversationId, recipientId, bot.UuidNewV4().String(), "PLAIN_TEXT", data, config.ClientId, config.SessionId, config.SessionKey)
}
