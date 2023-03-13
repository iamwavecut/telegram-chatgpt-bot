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
	case 5, 6:
		levelColor = gray
	case 3:
		levelColor = yellow
	case 2, 1, 0:
		levelColor = red
	case 4:
		levelColor = blue
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
	output = strings.Replace(output, "\r", "\\r", -1)
	output = strings.Replace(output, "\n", "\\n", -1) + "\n"
	return []byte(output), nil
}
