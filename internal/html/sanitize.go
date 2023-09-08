package html

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/iamwavecut/tool"
	"golang.org/x/net/html"
)

func Sanitize(input string, allowedTags []string) (string, error) {
	var output strings.Builder

	tokenizer := html.NewTokenizer(strings.NewReader(input))
	for {
		tokenType := tokenizer.Next()
		token := tokenizer.Token()

		switch tokenType {
		case html.ErrorToken: // End of the document
			if !errors.Is(tokenizer.Err(), io.EOF) {
				return output.String(), tokenizer.Err()
			}
			return output.String(), nil
		case html.TextToken:
			output.WriteString(html.EscapeString(token.Data))
		case html.StartTagToken, html.EndTagToken, html.SelfClosingTagToken, html.CommentToken, html.DoctypeToken:
			if tool.In(token.Data, allowedTags...) {
				tag := token.String()
				tag = strings.ReplaceAll(tag, "&", "&amp;")
				output.WriteString(tag)
			} else {
				tag := token.Data
				if tokenType == html.EndTagToken {
					tag = "/" + tag
				}
				output.WriteString(fmt.Sprintf("&lt;%s&gt;", tag))
			}
		}
	}
}
