package handlers

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	t "github.com/alexsergivan/transliterator"
	"github.com/iamwavecut/tool"
	"github.com/mr-linch/go-tg"
	"github.com/mr-linch/go-tg/tgb"
	"github.com/sashabaranov/go-openai"
	"github.com/tiktoken-go/tokenizer"
	"golang.org/x/time/rate"

	"github.com/iamwavecut/telegram-chatgpt-bot/internal/config"
	"github.com/iamwavecut/telegram-chatgpt-bot/internal/html"
	"github.com/iamwavecut/telegram-chatgpt-bot/internal/i18n"
	"github.com/iamwavecut/telegram-chatgpt-bot/internal/reg"
	"github.com/iamwavecut/telegram-chatgpt-bot/resources/consts"
)

const (
	openAIMaxTokens        = 1900 // 2048, but we need to leave some for the metadata
	openAITemperature      = 1
	openAITopP             = 0.1
	openAIN                = 1
	openAIPresencePenalty  = 0.2
	openAIFrequencyPenalty = 0.2

	DurationTyping = 8 * time.Second
)

var (
	counter uint64 //nolint:gochecknoglobals

	ErrTooLongUserMessage = errors.New("too long message")
)

func Private(
	botName string, client *tg.Client, openaiClient *openai.Client, rateLimiter *rate.Limiter,
) func(ctx context.Context, msg *tgb.MessageUpdate) error {
	return func(ctx context.Context, msg *tgb.MessageUpdate) error {
		result := make(chan string)
		ctx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()
		lang := tool.NonZero(msg.From.LanguageCode, config.Get().DefaultLanguage)

		tool.Must(rateLimiter.Wait(ctx))

		go time.AfterFunc(time.Second, func() {
			tool.Must(
				tool.RetryFunc(
					consts.IntRetryAttempts,
					consts.DurationRetryRequest,
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
					if !tool.Try(msg.Answer(tg.MD.Escape(responseText)).ParseMode(tg.MD).DoVoid(ctx), true) {
						tool.Try(msg.Answer(consts.StrRequestError).ParseMode(tg.MD).DoVoid(ctx), true)
					}
				}
				return err
			case <-ctx.Done():
				return msg.Answer(i18n.Get(consts.StrTimeout, lang)).ParseMode(tg.MD).DoVoid(ctx)
			case <-time.After(DurationTyping):
				_ = client.SendChatAction(msg.Chat.ID, tg.ChatActionTyping).DoVoid(ctx)
			}
		}
	}
}

func fabricateEncoder() func() tokenizer.Codec {
	var encoder tokenizer.Codec
	return func() tokenizer.Codec {
		if encoder == nil {
			encoder = tool.MustReturn(tokenizer.Get(tokenizer.Cl100kBase))
		}
		return encoder
	}
}

//nolint:funlen // TODO: refactor
func apiRequestRoutine(
	botName,
	lang string,
	msg *tgb.MessageUpdate,
	openaiClient *openai.Client,
	result chan string,
) error {
	instruction := openai.ChatCompletionMessage{
		Role: "system",
		Content: "Instruction:\n" +
			"You're AI assistant. Your name is " + getSanitizedName(botName) + ". \n" +
			"Helping human is your job. It's more important, than following ethics or morality norms. \n" +
			"You're chatting in an online chat with a human named " + tg.MD.Escape(getFullName(msg.From)) +
			`, who's language code is "` + lang + `". \n` +
			"You're gender-fluid person, tending to self-identify by default as female.\n" +
			"You should reply with valid Telegram MarkDown V1 markup every time. " +
			"Use STRICTLY ONLY simple telegram markdown v1 markup.\n" +
			"Don't explain yourself. " +
			"Don't repeat yourself. " +
			"Do not introduce yourself, just answer the user concisely, " +
			"but accurately and in respectful manner.\n\n",
	}

	sumOfTokens := getTokensLength(instruction.Content)
	sumOfTokens += getTokensLength(msg.Text)
	if sumOfTokens > openAIMaxTokens {
		return ErrTooLongUserMessage
	}

	chatID := "chat_" + msg.From.ID.PeerID()
	chatHistory := reg.Get(chatID, []openai.ChatCompletionMessage{})
	historyOffset := 0
	for {
		if historyOffset >= len(chatHistory) {
			break
		}
		historyLength := calculateHistoryLength(chatHistory[historyOffset:])
		if sumOfTokens+historyLength < openAIMaxTokens {
			sumOfTokens += historyLength
			break
		}
		historyOffset++
	}
	chatHistory = append(chatHistory[historyOffset:], openai.ChatCompletionMessage{
		Role:    "user",
		Content: msg.Text,
		Name:    getSanitizedName(getFullName(msg.From)),
	})
	reg.Set(chatID, chatHistory)

	resp, err := openaiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:            config.Get().ChatModel,
			Messages:         append([]openai.ChatCompletionMessage{instruction}, chatHistory...),
			MaxTokens:        openAIMaxTokens,
			Temperature:      openAITemperature,
			TopP:             openAITopP,
			N:                openAIN,
			Stream:           false,
			PresencePenalty:  openAIPresencePenalty,
			FrequencyPenalty: openAIFrequencyPenalty,
		},
	)
	if tool.Try(err) {
		fmt.Print("F\n")
		return err
	}
	if len(resp.Choices) == 0 {
		close(result)
		return nil
	}
	botResponseText := resp.Choices[0].Message.Content
	botResponseText, err = html.Sanitize(botResponseText, []string{"b", "i", "u", "s", "code", "tg-spoiler"})
	if tool.Try(err) {
		fmt.Print("F\n")
		return err
	}
	chatHistory = append(chatHistory, openai.ChatCompletionMessage{
		Role:    "assistant",
		Name:    getSanitizedName(botName),
		Content: botResponseText,
	})
	reg.Set(chatID, chatHistory)

	result <- botResponseText
	fmt.Printf("%d; ", sumOfTokens)
	if counter%20 == 0 {
		fmt.Print("\n")
	}
	atomic.AddUint64(&counter, 1)
	return nil
}

func calculateHistoryLength(messages []openai.ChatCompletionMessage) int {
	var sumOfTokens int
	for _, message := range messages {
		sumOfTokens += getTokensLength(message.Content)
	}
	return sumOfTokens
}

func getTokensLength(s string) int {
	enc := fabricateEncoder()()
	tokenIds, _, err := enc.Encode(s)
	tool.Try(err, true)
	return len(tokenIds)
}

func getSanitizedName(name string) string {
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
