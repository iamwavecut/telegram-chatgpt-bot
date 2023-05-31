package handlers

import (
	"context"

	"github.com/iamwavecut/telegram-chatgpt-bot/internal/reg"
	"github.com/mr-linch/go-tg/tgb"
)

func Reset() func(ctx context.Context, msg *tgb.MessageUpdate) error {
	return func(ctx context.Context, msg *tgb.MessageUpdate) error {
		reg.Delete("chat_" + msg.From.ID.PeerID())
		return nil
	}
}
