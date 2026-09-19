package prompts

// WithScopedTools appends langchaingo's MRKL tool-descriptions placeholder to
// a system prompt, phrased as an available-but-optional toolset. Used by the
// generation agents, which may finish without calling every scoped tool.
func WithScopedTools(systemPrompt string) string {
	return systemPrompt + "\n\nYou may use these scoped tools when needed:\n{{.tool_descriptions}}"
}

// WithOnlyScopedTools is the same suffix phrased as an exclusive toolset.
// Used by the Job Hunter and Job Import agents, which must stay within their
// small, fixed tool set.
func WithOnlyScopedTools(systemPrompt string) string {
	return systemPrompt + "\n\nYou may use only these scoped tools:\n{{.tool_descriptions}}"
}
