package memtool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
)

// Run appends userText to history as a new user turn, then drives the
// memory tool_use / tool_result cycle until the model replies without
// calling the tool. It returns the full updated history (safe to pass back
// into a later Run call) and the model's final message.
func (l *Loop) Run(ctx context.Context, history []anthropic.BetaMessageParam, userText string) ([]anthropic.BetaMessageParam, *anthropic.BetaMessage, error) {
	messages := append(append([]anthropic.BetaMessageParam{}, history...), anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(userText)))

	memoryTool := anthropic.BetaToolUnionParam{OfMemoryTool20250818: &anthropic.BetaMemoryTool20250818Param{}}

	for iteration := 1; ; iteration++ {
		if l.MaxIterations > 0 && iteration > l.MaxIterations {
			return messages, nil, fmt.Errorf("memtool: exceeded %d iterations without a final answer", l.MaxIterations)
		}

		resp, err := l.Client.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
			Model:     l.Model,
			MaxTokens: l.MaxTokens,
			Messages:  messages,
			Tools:     []anthropic.BetaToolUnionParam{memoryTool},
			Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaContextManagement2025_06_27},
		})
		if err != nil {
			return messages, nil, fmt.Errorf("memtool: message create: %w", err)
		}
		messages = append(messages, resp.ToParam())

		var toolUses []anthropic.BetaToolUseBlock
		for _, block := range resp.Content {
			if block.Type == "tool_use" {
				toolUses = append(toolUses, block.AsToolUse())
			}
		}
		if len(toolUses) == 0 {
			return messages, resp, nil
		}

		results := make([]anthropic.BetaContentBlockParamUnion, len(toolUses))
		for i, tu := range toolUses {
			results[i] = l.executeOne(tu)
		}
		messages = append(messages, anthropic.NewBetaUserMessage(results...))
	}
}

// executeOne routes a single tool_use block through the Executor. The
// memory tool's name is fixed to "memory" by the API, so anything else
// means a different tool was wired into the same request by mistake.
func (l *Loop) executeOne(tu anthropic.BetaToolUseBlock) anthropic.BetaContentBlockParamUnion {
	if tu.Name != "memory" {
		return anthropic.NewBetaToolResultBlock(tu.ID, fmt.Sprintf("unknown tool %q", tu.Name), true)
	}
	inputBytes, err := json.Marshal(tu.Input)
	if err != nil {
		return anthropic.NewBetaToolResultBlock(tu.ID, fmt.Sprintf("marshal tool input: %v", err), true)
	}
	var cmd anthropic.BetaMemoryTool20250818CommandUnion
	if err := json.Unmarshal(inputBytes, &cmd); err != nil {
		return anthropic.NewBetaToolResultBlock(tu.ID, fmt.Sprintf("unmarshal memory command: %v", err), true)
	}
	result, isError := l.Executor.Execute(cmd)
	return anthropic.NewBetaToolResultBlock(tu.ID, result, isError)
}

// Loop drives a Claude conversation with Anthropic's native memory tool
// enabled, routing every memory tool_use block through an Executor until
// the model stops calling the tool. The memory tool has no input_schema of
// its own (Anthropic supplies it server-side), so it cannot use the SDK's
// generic BetaTool/BetaToolRunner abstraction, which always requires one —
// this is the hand-rolled equivalent for that one tool.
type Loop struct {
	Client        anthropic.Client
	Model         anthropic.Model
	MaxTokens     int64
	Executor      *Executor
	MaxIterations int
}
