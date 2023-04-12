package handlers

import (
	"context"
	"fmt"

	"github.com/iamwavecut/tool"
	"github.com/mr-linch/go-tg"
	"github.com/mr-linch/go-tg/tgb"

	"github.com/iamwavecut/telegram-chatgpt-bot/internal/config"
	"github.com/iamwavecut/telegram-chatgpt-bot/internal/i18n"
	"github.com/iamwavecut/telegram-chatgpt-bot/internal/reg"
	"github.com/iamwavecut/telegram-chatgpt-bot/resources/consts"
)

func Start(botName string) func(ctx context.Context, msg *tgb.MessageUpdate) error {
	return func(ctx context.Context, msg *tgb.MessageUpdate) error {
		lang := tool.NonZero(msg.From.LanguageCode, config.Get().DefaultLanguage)
		chatID := "chat_" + msg.From.ID.PeerID()
		reg.Delete(chatID)

		return msg.Answer(
			tg.MD.Text(
				tg.MD.Bold(fmt.Sprintf(i18n.Get(consts.StrHello, lang), botName)),
				"",
				tg.MD.Line(
					fmt.Sprintf(
						i18n.Get(consts.StrIntro, lang),
						tg.MD.Link(
							i18n.Get("here", lang),
							"github.com/iamwavecut/telegram-chatgpt-bot",
						),
					),
				),
				"",
				tg.MD.Italic(
					i18n.Get(consts.StrOutro, lang),
				),
			),
		).ParseMode(tg.MD).DoVoid(ctx)
	}
}
