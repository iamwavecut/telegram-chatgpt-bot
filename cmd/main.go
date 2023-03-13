package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	t "github.com/alexsergivan/transliterator"
	"github.com/iamwavecut/tool"
	"github.com/mr-linch/go-tg"
	"github.com/mr-linch/go-tg/tgb"
	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"

	"github.com/iamwavecut/telegram-chatgpt-bot/internal/config"
	"github.com/iamwavecut/telegram-chatgpt-bot/internal/i18n"
	"github.com/iamwavecut/telegram-chatgpt-bot/internal/infra"
	"github.com/iamwavecut/telegram-chatgpt-bot/internal/reg"
)

func main() {
	ctx := context.Background()

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill, syscall.SIGTERM)
	defer cancel()

	go func() {
		if err := run(ctx); err != nil {
			fmt.Println(err)
			defer os.Exit(1)
		}
	}()

	<-infra.MonitorExecutable()
	log.Errorln("executable file was modified")
	os.Exit(0)
}

func run(ctx context.Context) error {
	client := tg.New(config.Get().TelegramAPIToken)

	me := tool.MustReturn(client.GetMe().Do(ctx))
	botName := tg.MD.Escape(me.FirstName)

	openaiClient := openai.NewClient(config.Get().OpenAIToken)

	router := tgb.NewRouter().
		Message(
			handleStart(botName),
			tgb.Command("start", tgb.WithCommandAlias("help")),
			tgb.ChatType(tg.ChatTypePrivate),
		).
		Message(
			handlePrivate(botName, client, openaiClient),
			tgb.ChatType(tg.ChatTypePrivate),
		).
		Message(
			handlePublic(&me),
			tgb.ChatType(tg.ChatTypeGroup, tg.ChatTypeSupergroup),
			tgb.Regexp(regexp.MustCompile("(?mi)("+me.FirstName+"|/start|/start"+me.Username.PeerID()+")")),
		)
	tool.Console("starting")
	return tgb.NewPoller(
		router,
		client,
		tgb.WithPollerRetryAfter(time.Minute),
	).Run(ctx)
}

func handleStart(botName string) func(ctx context.Context, msg *tgb.MessageUpdate) error {
	return func(ctx context.Context, msg *tgb.MessageUpdate) error {
		tool.Console("start")
		lang := tool.NonZero(msg.From.LanguageCode, config.Get().DefaultLanguage)
		chatID := "chat_" + msg.From.ID.PeerID()
		reg.Delete(chatID)

		return msg.Answer(
			tg.MD.Text(
				tg.MD.Bold(fmt.Sprintf(i18n.Get("Hi, my name is %s!", lang), botName)),
				"",
				tg.MD.Line(
					fmt.Sprintf(
						i18n.Get("I'm a bot, based on ChatGPT API. My source code is available %s, for you to audit it. I do not log nor store your message, but keep in mind, that I should store some history at runtime, to keep context, and I send it to OpenAI API. If it's concerning you – please stop and delete me.", lang),
						tg.MD.Link(
							i18n.Get("here", lang),
							"github.com//iamwavecut/telegram-chatgpt-bot",
						),
					),
				),
				"",
				tg.MD.Italic(
					i18n.Get("If you want to restart the conversation from scratch, just type /start and the bot's recent memories will fade away.", lang),
				),
			),
		).ParseMode(tg.MD).DoVoid(ctx)
	}
}

func handlePrivate(botName string, client *tg.Client, openaiClient *openai.Client) func(ctx context.Context, msg *tgb.MessageUpdate) error {
	return func(ctx context.Context, msg *tgb.MessageUpdate) error {
		tool.Console("answer")
		result := make(chan string)
		ctx, cancel := context.WithTimeout(ctx, time.Second*60)
		defer cancel()
		lang := tool.NonZero(msg.From.LanguageCode, config.Get().DefaultLanguage)
		go func() {
			chatID := "chat_" + msg.From.ID.PeerID()
			chatHistory := reg.Get(chatID, []openai.ChatCompletionMessage{
				{
					Role: "system",
					Content: "Instruction:\n" +
						"Your name is " + sanitizeName(botName) + ". \n" +
						"You're chatting in an online chat with a human named " + sanitizeName(getFullName(msg.From)) + ", who's language code is \"" + lang + "\". \n" +
						"You're genderfluent person\n" +
						"Do not introduce yourself, just answer the user.\n\n",
				},
			})
			chatHistory = append(chatHistory, openai.ChatCompletionMessage{
				Role:    "user",
				Content: msg.Text,
				Name:    sanitizeName(getFullName(msg.From)),
			})
			reg.Set(chatID, chatHistory)

			resp := tool.MustReturn(
				openaiClient.CreateChatCompletion(
					context.Background(),
					openai.ChatCompletionRequest{
						Model:            openai.GPT3Dot5Turbo,
						Messages:         chatHistory,
						MaxTokens:        500,
						Temperature:      1,
						TopP:             .1,
						N:                1,
						Stream:           false,
						PresencePenalty:  0.2,
						FrequencyPenalty: 0.2,
					},
				),
			)
			if len(resp.Choices) == 0 {
				close(result)
			}
			botResponseText := resp.Choices[0].Message.Content
			chatHistory = append(chatHistory, openai.ChatCompletionMessage{
				Role:    "assistant",
				Name:    sanitizeName(botName),
				Content: botResponseText,
			})
			if len(chatHistory) > 10 {
				chatHistory = chatHistory[len(chatHistory)-10:]
			}
			reg.Set(chatID, chatHistory)
			result <- botResponseText
			return
		}()

		_ = client.SendChatAction(msg.Chat.ID, tg.ChatActionTyping).DoVoid(ctx)
		for {
			select {
			case responseText, isOpen := <-result:
				if !isOpen && responseText == "" {
					responseText = i18n.Get("Sorry, I don't have an answer.", lang)
				}
				tool.Console(responseText)
				err := msg.Answer(responseText).ParseMode(tg.MD).DoVoid(ctx)
				if tool.Try(err) {
					tool.Console(err)
				}
			case <-ctx.Done():
				return msg.Answer(i18n.Get("I'm sorry, but this takes an unacceptable duration of time to answer. Request aborted.", lang)).ParseMode(tg.MD).DoVoid(ctx)
			case <-time.After(time.Second * 8):
				_ = client.SendChatAction(msg.Chat.ID, tg.ChatActionTyping).DoVoid(ctx)
			}
		}
	}
}

func handlePublic(me *tg.User) func(ctx context.Context, msg *tgb.MessageUpdate) error {
	return func(ctx context.Context, msg *tgb.MessageUpdate) error {
		tool.Console("public")
		lang := tool.NonZero(msg.From.LanguageCode, config.Get().DefaultLanguage)

		layout := tg.NewButtonLayout[tg.InlineKeyboardButton](1).Row(
			tg.NewInlineKeyboardButtonURL(i18n.Get("Switch to private chat", lang), me.Username.Link()+"?start=start"),
		)
		return msg.Answer(
			i18n.Get("Unfortunately, I work terrible in groups, as ChatGPT was designed to be used in dialogues. Please message me in private.", msg.From.LanguageCode),
		).
			ParseMode(tg.MD).
			ReplyMarkup(tg.NewInlineKeyboardMarkup(layout.Keyboard()...)).
			DoVoid(ctx)
	}
}

func sanitizeName(name string) string {
	return reg.Get("name_"+name, func() string {
		name = t.NewTransliterator(nil).Transliterate(strings.ToLower(name), "en")
		re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
		name = re.ReplaceAllString(name, "")
		if len(name) > 64 {
			name = name[:64]
		}
		return name
	}())
}

func getFullName(user *tg.User) string {
	userName := user.FirstName + " " + user.LastName
	userName = strings.TrimSpace(userName)
	if 0 == len(userName) {
		userName = user.Username.PeerID()
	}
	if 0 == len(userName) || userName == "@" {
		userName = user.ID.PeerID()
	}
	return userName
}
