package config

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

type NbFormatter struct{}

func (f *NbFormatter) Format(entry *log.Entry) ([]byte, error) {
	const (
		red    = 31
		yellow = 33
		blue   = 36
		gray   = 37
	)
	levelColor := blue
	switch entry.Level {
	case log.DebugLevel, log.TraceLevel:
		levelColor = gray
	case log.InfoLevel:
		levelColor = blue
	case log.WarnLevel:
		levelColor = yellow
	case log.ErrorLevel, log.FatalLevel, log.PanicLevel:
		levelColor = red
	}
	level := fmt.Sprintf(
		"\x1b[%dm%s\x1b[0m",
		levelColor,
		strings.ToUpper(entry.Level.String())[:4],
	)

	output := "level=" + level
	output += " ts=" + entry.Time.Format("2006-01-02 15:04:05.000")
	for k, val := range entry.Data {
		var s string
		if m, err := json.Marshal(val); err == nil {
			s = string(m)
		}
		if s == "" {
			continue
		}
		output += fmt.Sprintf(" %s=%s", k, s)
	}
	output += ` msg="` + entry.Message + `"`
	output = strings.ReplaceAll(output, "\r", "\\r")
	output = strings.ReplaceAll(output, "\n", "\\n") + "\n"
	return []byte(output), nil
}
