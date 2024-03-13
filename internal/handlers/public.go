package handlers

import (
	"context"

	"github.com/iamwavecut/tool"
	"github.com/mr-linch/go-tg"
	"github.com/mr-linch/go-tg/tgb"

	"github.com/iamwavecut/telegram-chatgpt-bot/internal/config"
	"github.com/iamwavecut/telegram-chatgpt-bot/internal/i18n"
	"github.com/iamwavecut/telegram-chatgpt-bot/resources/consts"
)

func Public(me *tg.User) func(ctx context.Context, msg *tgb.MessageUpdate) error {
	return func(ctx context.Context, msg *tgb.MessageUpdate) error {
		lang := tool.NonZero(msg.From.LanguageCode, config.Get().DefaultLanguage)

		layout := tg.NewButtonLayout[tg.InlineKeyboardButton](1).Row(
			tg.NewInlineKeyboardButtonURL(i18n.Get("Switch to private chat", lang), me.Username.Link()+"?start=start"),
		)
		return msg.Answer(
			i18n.Get(consts.StrNoPublic, msg.From.LanguageCode),
		).
			ParseMode(tg.MD).
			ReplyParameters(tg.ReplyParameters{
				MessageID: msg.Message.ID,
				ChatID:    msg.Message.Chat.ID,

				AllowSendingWithoutReply: true,
			}).
			ReplyMarkup(tg.NewInlineKeyboardMarkup(layout.Keyboard()...)).
			DoVoid(ctx)
	}
}
