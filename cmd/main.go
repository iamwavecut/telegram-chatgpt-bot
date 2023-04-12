package main

import (
	"context"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/iamwavecut/tool"
	"github.com/mr-linch/go-tg"
	"github.com/mr-linch/go-tg/tgb"
	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"

	"github.com/iamwavecut/telegram-chatgpt-bot/internal/config"
	"github.com/iamwavecut/telegram-chatgpt-bot/internal/handlers"
	"github.com/iamwavecut/telegram-chatgpt-bot/internal/infra"
	"github.com/iamwavecut/telegram-chatgpt-bot/resources/consts"
)

func main() {
	ctx := context.Background()

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill, syscall.SIGTERM)

	go func() {
		if err := run(ctx); err != nil {
			log.Errorln(err)
			cancel()
		}
	}()

	<-infra.MonitorExecutable()
	log.Errorln("executable file was modified")
	cancel()
}

func run(ctx context.Context) error {
	rateLimiter := rate.NewLimiter(rate.Every(consts.MinTimeBetweenRequests), 1)
	client := tg.New(config.Get().TelegramAPIToken)

	me := tool.MustReturn(client.GetMe().Do(ctx))
	botName := tg.MD.Escape(me.FirstName)

	openaiClient := openai.NewClient(config.Get().OpenAIToken)

	router := tgb.NewRouter().
		Message(
			handlers.Start(botName),
			tgb.Command("start", tgb.WithCommandAlias("help")),
			tgb.ChatType(tg.ChatTypePrivate),
		).
		Message(
			handlers.Private(botName, client, openaiClient, rateLimiter),
			tgb.ChatType(tg.ChatTypePrivate),
		).
		Message(
			handlers.Public(&me),
			tgb.ChatType(tg.ChatTypeGroup, tg.ChatTypeSupergroup),
			tgb.Regexp(regexp.MustCompile("(?mi)(^"+me.FirstName+"|/start|/start"+me.Username.PeerID()+")")),
		)
	tool.Console("started")
	return tgb.NewPoller(
		router,
		client,
		tgb.WithPollerRetryAfter(time.Minute),
	).Run(ctx)
}
