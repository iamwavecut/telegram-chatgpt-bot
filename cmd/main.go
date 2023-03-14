package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
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

const (
	StrHello = "Hi, my name is %s!"
	StrIntro = "I'm a bot, based on ChatGPT API. My source code " +
		"is available %s, for you to audit it. I do not " +
		"log nor store your message, but keep in mind, that " +
		"I should store some history at runtime, to keep context, " +
		"and I send it to OpenAI API. If it's concerning " +
		"you – please stop and delete me."
	StrOutro = "If you want to restart the conversation from " +
		"scratch, just type /start and the bot's " +
		"recent memories will fade away."
	StrTimeout = "I'm sorry, but this takes an unacceptable " +
		"duration of time to answer. Request aborted."
	StrNoPublic = "Unfortunately, I work terrible in groups, " +
		"as ChatGPT was designed to be used in dialogues. " +
		"Please message me in private."
	StrRequestError = "Unfortunately, there was an error during the request. " +
		"Please try again later. If it doesn't help, " +
		"please contact the bot's maintainer."

	DurationTyping       = 8 * time.Second
	DurationRetryRequest = 5 * time.Second

	openaiMaxTokens        = 500
	openaiTemperature      = 1
	openaiTopP             = 0.1
	openaiN                = 1
	openaiPresencePenalty  = 0.2
	openaiFrequencyPenalty = 0.2

	IntChatHistoryLength = 10
	IntRetryAttempts     = 5
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
	tool.Console("started")
	return tgb.NewPoller(
		router,
		client,
		tgb.WithPollerRetryAfter(time.Minute),
	).Run(ctx)
}

func handleStart(botName string) func(ctx context.Context, msg *tgb.MessageUpdate) error {
	return func(ctx context.Context, msg *tgb.MessageUpdate) error {
		lang := tool.NonZero(msg.From.LanguageCode, config.Get().DefaultLanguage)
		chatID := "chat_" + msg.From.ID.PeerID()
		reg.Delete(chatID)

		return msg.Answer(
			tg.MD.Text(
				tg.MD.Bold(fmt.Sprintf(i18n.Get(StrHello, lang), botName)),
				"",
				tg.MD.Line(
					fmt.Sprintf(
						i18n.Get(StrIntro, lang),
						tg.MD.Link(
							i18n.Get("here", lang),
							"github.com//iamwavecut/telegram-chatgpt-bot",
						),
					),
				),
				"",
				tg.MD.Italic(
					i18n.Get(StrOutro, lang),
				),
			),
		).ParseMode(tg.MD).DoVoid(ctx)
	}
}

var lock = sync.Mutex{} //nolint:gochecknoglobals // lock is a global lock

func handlePrivate(
	botName string, client *tg.Client, openaiClient *openai.Client,
) func(ctx context.Context, msg *tgb.MessageUpdate) error {
	return func(ctx context.Context, msg *tgb.MessageUpdate) error {
		result := make(chan string)
		ctx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()
		lang := tool.NonZero(msg.From.LanguageCode, config.Get().DefaultLanguage)
		lock.Lock()
		defer lock.Unlock()

		go time.AfterFunc(time.Second, func() {
			tool.Must(
				tool.RetryFunc(
					IntRetryAttempts,
					DurationRetryRequest,
					func() error { return apiRequestRoutine(botName, lang, msg, openaiClient, result) },
				),
			)
		})

		_ = client.SendChatAction(msg.Chat.ID, tg.ChatActionTyping).DoVoid(ctx)
		for {
			select {
			case responseText, isOpen := <-result:
				if !isOpen && responseText == "" {
					responseText = i18n.Get("Sorry, I don't have an answer.", lang)
				}
				err := msg.Answer(responseText).ParseMode(tg.MD).DoVoid(ctx)
				if tool.Try(err) {
					tool.Console(err, responseText)
					tool.Try(msg.Answer(StrRequestError).ParseMode(tg.MD).DoVoid(ctx), true)
				}
				return err
			case <-ctx.Done():
				return msg.Answer(i18n.Get(StrTimeout, lang)).ParseMode(tg.MD).DoVoid(ctx)
			case <-time.After(DurationTyping):
				_ = client.SendChatAction(msg.Chat.ID, tg.ChatActionTyping).DoVoid(ctx)
			}
		}
	}
}

func apiRequestRoutine(
	botName,
	lang string,
	msg *tgb.MessageUpdate,
	openaiClient *openai.Client,
	result chan string,
) error {
	chatID := "chat_" + msg.From.ID.PeerID()
	chatHistory := reg.Get(chatID, []openai.ChatCompletionMessage{})
	chatHistory = append(chatHistory, openai.ChatCompletionMessage{
		Role:    "user",
		Content: msg.Text,
		Name:    sanitizeName(getFullName(msg.From)),
	})
	if len(chatHistory) > IntChatHistoryLength {
		chatHistory = chatHistory[len(chatHistory)-IntChatHistoryLength:]
	}
	reg.Set(chatID, chatHistory)

	resp, err := openaiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: append([]openai.ChatCompletionMessage{
				{
					Role: "system",
					Content: "Instruction:\n" +
						"You're AI assistant. Your name is " + sanitizeName(botName) + ". \n" +
						"You're chatting in an online chat with a human named " +
						sanitizeName(getFullName(msg.From)) +
						`, who's language code is "` + lang + `". \n` +
						"You're genderfluid person, tending to self-identify mostly as female.\n" +
						"You should send valid markdown text every time.\n" +
						"Do not introduce yourself, just answer the user concisely.\n\n",
				},
			}, chatHistory...),
			MaxTokens:        openaiMaxTokens,
			Temperature:      openaiTemperature,
			TopP:             openaiTopP,
			N:                openaiN,
			Stream:           false,
			PresencePenalty:  openaiPresencePenalty,
			FrequencyPenalty: openaiFrequencyPenalty,
		},
	)
	if tool.Try(err) {
		return err
	}
	if len(resp.Choices) == 0 {
		close(result)
		return nil
	}
	botResponseText := resp.Choices[0].Message.Content
	chatHistory = append(chatHistory, openai.ChatCompletionMessage{
		Role:    "assistant",
		Name:    sanitizeName(botName),
		Content: botResponseText,
	})
	reg.Set(chatID, chatHistory)

	result <- botResponseText
	return nil
}

func handlePublic(me *tg.User) func(ctx context.Context, msg *tgb.MessageUpdate) error {
	return func(ctx context.Context, msg *tgb.MessageUpdate) error {
		lang := tool.NonZero(msg.From.LanguageCode, config.Get().DefaultLanguage)

		layout := tg.NewButtonLayout[tg.InlineKeyboardButton](1).Row(
			tg.NewInlineKeyboardButtonURL(i18n.Get("Switch to private chat", lang), me.Username.Link()+"?start=start"),
		)
		return msg.Answer(
			i18n.Get(StrNoPublic, msg.From.LanguageCode),
		).
			ParseMode(tg.MD).
			ReplyMarkup(tg.NewInlineKeyboardMarkup(layout.Keyboard()...)).
			DoVoid(ctx)
	}
}

func sanitizeName(name string) string {
	const openaiMaxNameLen = 64
	return reg.Get("name_"+name, func() string {
		name = t.NewTransliterator(nil).Transliterate(strings.ToLower(name), "en")
		re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
		name = re.ReplaceAllString(name, "")
		if len(name) > openaiMaxNameLen {
			name = name[:64]
		}
		return name
	}())
}

func getFullName(user *tg.User) string {
	userName := user.FirstName + " " + user.LastName
	userName = strings.TrimSpace(userName)
	if len(userName) == 0 {
		userName = user.Username.PeerID()
	}
	if len(userName) == 0 || userName == "@" {
		userName = user.ID.PeerID()
	}
	return userName
}
