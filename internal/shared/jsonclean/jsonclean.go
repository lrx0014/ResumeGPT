// Package jsonclean extracts a JSON object from model output that may be
// wrapped in a markdown code fence or preceded/followed by stray text.
package jsonclean

import "strings"

// ExtractJSONObject strips a leading/trailing ```json or ``` fence and
// returns the substring between the outermost { and }, if present.
func ExtractJSONObject(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "```json")
	value = strings.TrimPrefix(value, "```")
	value = strings.TrimSuffix(value, "```")
	start, end := strings.Index(value, "{"), strings.LastIndex(value, "}")
	if start >= 0 && end > start {
		value = value[start : end+1]
	}
	return strings.TrimSpace(value)
}
