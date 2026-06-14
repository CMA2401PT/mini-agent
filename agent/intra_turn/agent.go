package intra_turn

import (
	"context"
	"fmt"

	"mini_agent/core"
)

// DefaultInterruptedToolResult is the default placeholder for tool calls whose
// execution was interrupted before a result landed.
const DefaultInterruptedToolResult = "[no result: the previous turn was interrupted before this tool call completed]"

// RunTurn runs a complete multi-turn tool-calling loop for one user input.
// It returns an OutStream of NotifyOrSyncEvents; range over it to observe phase
// notifications and apply turn sync primitives. history must already contain the
// user's prompt as the last turn. The history slice is mutated in place as the
// turn progresses.
func RunTurn(
	ctx context.Context,
	provider core.Provider,
	tools core.ToolSetAndRunner,
	history []core.Turn,
	interruptedResultPlaceHolder string,
) core.OutStream[core.ConversationOutput] {
	out := make(chan core.ConversationOutput)
	go runTurn(ctx, provider, tools, history, interruptedResultPlaceHolder, out)
	return out
}

func runTurn(
	ctx context.Context,
	provider core.Provider,
	tools core.ToolSetAndRunner,
	history []core.Turn,
	interruptedResultPlaceHolder string,
	out chan<- core.ConversationOutput,
) {
	lastIdx := len(history) - 1

	// Shared across inner loop: written by error returns, read by defer.
	var interruptErr error

	defer close(out)
	defer func() {
		if interruptedResultPlaceHolder == "" {
			interruptedResultPlaceHolder = DefaultInterruptedToolResult
		}
		msgs := PatchIncompleteCalls(history[lastIdx], interruptedResultPlaceHolder)
		if interruptErr != nil || len(msgs) > 0 {
			out <- core.ConversationOutput{
				BeforeEvent: core.KeyNotifyFailure{Err: interruptErr},
				SyncPrimitives: core.SyncPrimitiveAppendMessages{
					Messages: msgs,
				},
			}
		}
		if len(msgs) > 0 {
			history[lastIdx] = append(history[lastIdx], msgs...)
		}
	}()

	for {
		// --- request ---
		out <- core.ConversationOutput{BeforeEvent: core.KeyNotifyRequestSent{}}
		req := core.Request{
			Messages: TurnsToMessagesWithoutTransparent(history),
			Tools:    tools.ToolSchemas(),
		}
		ch, err := provider.Stream(ctx, req)
		if err != nil {
			interruptErr = fmt.Errorf("stream: %w", err)
			return
		}

		// --- deltas ---
		var msg core.AssistantMsg
		msgIdx := len(history[lastIdx])
		history[lastIdx] = append(history[lastIdx], msg)
		blockCount := 0
		reasoning := false
		output := false
		for chunk := range ch {
			blockCount += 1
			if chunk.Err != nil {
				interruptErr = fmt.Errorf("chunk: %w", chunk.Err)
				return
			}
			msg = msg.IncreaseFromChunk(chunk)
			history[lastIdx][msgIdx] = msg
			if blockCount == 1 {
				withReasoning := chunk.ReasoningActive || chunk.Reasoning != ""
				if withReasoning {
					reasoning = true
					out <- core.ConversationOutput{
						SyncPrimitives: core.SyncPrimitiveStartResponding{},
						AfterEvent:     core.KeyNotifyReasoningStart{},
					}
				} else {
					out <- core.ConversationOutput{
						SyncPrimitives: core.SyncPrimitiveStartResponding{},
						AfterEvent:     core.KeyNotifyOutputStart{},
					}
					output = true
				}
			} else if reasoning && chunk.Text != "" {
				reasoning = false
				out <- core.ConversationOutput{
					BeforeEvent: core.KeyNotifyReasoningEnd{},
					AfterEvent:  core.KeyNotifyOutputStart{},
				}
				output = true
			}

			out <- core.ConversationOutput{
				SyncPrimitives: core.SyncPrimitiveDeltaChunk{
					Chunk: chunk,
				},
			}
		}
		if output {
			out <- core.ConversationOutput{
				BeforeEvent: core.KeyNotifyOutputEnd{},
			}
		}
		// --- check for final answer ---
		calls := msg.ToolCalls
		if len(calls) == 0 {
			out <- core.ConversationOutput{
				BeforeEvent: core.KeyNotifyTurnEnd{},
			}
			return
		}

		out <- core.ConversationOutput{
			BeforeEvent: core.KeyNotifyToolUseStart{},
		}
		// --- run tools ---
		results, err := tools.RunCalls(ctx, calls)
		if err != nil {
			interruptErr = fmt.Errorf("run calls: %w", err)
			return
		}
		outResults := []core.Message{}
		for _, tr := range results {
			outResults = append(outResults, tr)
			history[lastIdx] = append(history[lastIdx], tr)
		}
		out <- core.ConversationOutput{
			SyncPrimitives: core.SyncPrimitiveAppendMessages{
				Messages: outResults,
			},
			BeforeEvent: core.KeyNotifyToolUseEnd{},
		}
	}
}

// TurnsToMessages flattens history turns into a single message list for the provider.
func TurnsToMessagesWithoutTransparent(history []core.Turn) []core.Message {
	n := 0
	for _, t := range history {
		n += len(t)
	}
	out := make([]core.Message, 0, n)
	for _, t := range history {
		for _, m := range t {
			if _, ok := m.(core.ProviderTransparentMessage); ok {
				continue
			}
			out = append(out, m)
		}
	}
	return out
}
