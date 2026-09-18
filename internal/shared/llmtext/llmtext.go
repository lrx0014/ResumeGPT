// Package llmtext provides helpers for working with langchaingo message content.
package llmtext

import (
	"strings"

	"github.com/tmc/langchaingo/llms"
)

// FlattenMessages joins the text parts of messages into a single prompt,
// separating parts with a blank line.
func FlattenMessages(messages []llms.MessageContent) string {
	var prompt strings.Builder
	for _, message := range messages {
		for _, part := range message.Parts {
			if text, ok := part.(llms.TextContent); ok {
				if prompt.Len() > 0 {
					prompt.WriteString("\n\n")
				}
				prompt.WriteString(text.Text)
			}
		}
	}
	return prompt.String()
}
