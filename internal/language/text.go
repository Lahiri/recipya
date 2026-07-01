package language

import (
	"strings"
	"unicode"
)

func normalizeText(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	text = strings.NewReplacer(
		"à", "a",
		"è", "e",
		"é", "e",
		"ì", "i",
		"ò", "o",
		"ù", "u",
		"'", " ",
		"’", " ",
		"`", " ",
	).Replace(text)

	var builder strings.Builder
	builder.Grow(len(text))
	lastSpace := true
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			builder.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(builder.String())
}
