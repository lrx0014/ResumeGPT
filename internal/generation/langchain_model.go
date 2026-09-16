package generation

import (
	"context"
	"strings"

	"github.com/lrx0014/ResumeGPT/internal/settings"
	"github.com/tmc/langchaingo/llms"
)

// gatewayModel adapts ResumeGPT's encrypted, provider-neutral connection runtime
// to LangChainGo's model interface.
type gatewayModel struct {
	gateway       Gateway
	runtime       settings.RuntimeConnection
	model         string
	systemPrompt  string
	images        []string
	imageSource   func() []string
	maxTokens     int
	agentOutput   bool
	requiredTools []requiredAgentTool
}

type requiredAgentTool struct {
	name      string
	satisfied func() bool
}

var _ llms.Model = (*gatewayModel)(nil)

func (m *gatewayModel) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	response, err := m.GenerateContent(ctx, []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, prompt)}, options...)
	if err != nil {
		return "", err
	}
	if len(response.Choices) == 0 {
		return "", ErrLLM
	}
	return response.Choices[0].Content, nil
}

func (m *gatewayModel) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	requiredTool := m.nextRequiredTool()
	if m.agentOutput && requiredTool == "read_template_source" {
		return &llms.ContentResponse{Choices: []*llms.ContentChoice{{Content: "Action: read_template_source\nAction Input: Inspect the complete template project before producing LaTeX."}}}, nil
	}
	if m.agentOutput && len(m.requiredTools) > 0 && requiredTool == "" {
		return &llms.ContentResponse{Choices: []*llms.ContentChoice{{Content: "Final Answer: Required tool validation completed successfully."}}}, nil
	}
	callOptions := llms.CallOptions{MaxTokens: m.maxTokens}
	for _, option := range options {
		option(&callOptions)
	}
	maxTokens := callOptions.MaxTokens
	if maxTokens <= 0 {
		maxTokens = m.maxTokens
	}
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
	images := m.images
	if m.imageSource != nil {
		images = m.imageSource()
	}
	content, err := m.gateway.Complete(ctx, m.runtime, m.model, m.systemPrompt, prompt.String(), images, maxTokens)
	if err != nil {
		return nil, err
	}
	// The MRKL executor requires explicit action and finish markers. A role may
	// also require a successful tool call before it is allowed to finish.
	if m.agentOutput && requiredTool != "" && !strings.Contains(content, "Action:") {
		candidate := strings.TrimSpace(content)
		if index := strings.LastIndex(candidate, "Final Answer:"); index >= 0 {
			candidate = strings.TrimSpace(candidate[index+len("Final Answer:"):])
		}
		content = "Action: " + requiredTool + "\nAction Input: " + candidate
	} else if m.agentOutput && !strings.Contains(content, "Final Answer:") && !strings.Contains(content, "Action:") {
		content = "Final Answer: " + content
	}
	return &llms.ContentResponse{Choices: []*llms.ContentChoice{{Content: content}}}, nil
}

func (m *gatewayModel) nextRequiredTool() string {
	for _, requirement := range m.requiredTools {
		if requirement.satisfied == nil || !requirement.satisfied() {
			return requirement.name
		}
	}
	return ""
}
