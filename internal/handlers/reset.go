package handlers

import (
	"context"

	"github.com/mr-linch/go-tg/tgb"

	"github.com/iamwavecut/telegram-chatgpt-bot/internal/reg"
)

func Reset() func(ctx context.Context, msg *tgb.MessageUpdate) error {
	return func(_ context.Context, msg *tgb.MessageUpdate) error {
		reg.Delete("chat_" + msg.From.ID.PeerID())
		return nil
	}
}
