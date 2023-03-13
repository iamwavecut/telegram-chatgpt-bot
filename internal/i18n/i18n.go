package i18n

import (
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/iamwavecut/telegram-chatgpt-bot/internal/config"
	"github.com/iamwavecut/telegram-chatgpt-bot/resources"

	log "github.com/sirupsen/logrus"
)

var state = struct {
	translations       map[string]map[string]string // [key][lang][translation]
	defaultLanguage    string
	availableLanguages []string
}{
	translations:       map[string]map[string]string{},
	defaultLanguage:    config.Get().DefaultLanguage,
	availableLanguages: []string{"en"},
}

func GetLanguagesList() []string {
	return state.availableLanguages[:]
}

var initialize sync.Once

func Get(key, lang string) string {
	initialize.Do(func() {
		if len(state.translations) > 0 {
			return
		}
		i18n, err := resources.FS.ReadFile("i18n.yaml")
		if err != nil {
			log.WithError(err).Errorln("cant load translations")
			return
		}
		if err := yaml.Unmarshal(i18n, &(state.translations)); err != nil {
			log.WithError(err).Errorln("cant unmarshal translations")
			return
		}
		languages := map[string]struct{}{}
		for _, langs := range state.translations {
			for lang := range langs {
				languages[strings.ToLower(lang)] = struct{}{}
			}
		}
		for lang := range languages {
			state.availableLanguages = append(state.availableLanguages, lang)
		}
		sort.Strings(state.availableLanguages)
		log.Traceln("languages count:", len(state.availableLanguages))
	})

	if "en" == lang {
		return key
	}
	if res, ok := state.translations[key][strings.ToUpper(lang)]; ok {
		return res
	}
	log.Traceln(`no "` + lang + `" translation for key "` + key + `"`)
	return key
}
