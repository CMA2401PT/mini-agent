package plain

import (
	"context"
	"errors"
	"sync"

	"mini_agent/agent/intra_turn"
	"mini_agent/core"
)

var errUserInterrupt = errors.New("interrupted by user")
var errStageTerminated = errors.New("stage terminated")

type PlainConversationCtrl struct {
	InitSystemPrompt       []core.Turn
	InterruptAppendMessage []core.Message
	Provider               core.Provider
	Tools                  core.ToolSetAndRunner
}

func (c *PlainConversationCtrl) Emit(
	ctx context.Context,
	initCmds []core.UserCommand,
	history []core.Turn,
) (core.ControlHandle, core.OutStream[core.ConversationOutput], error) {
	if history == nil {
		history = c.InitSystemPrompt
	}
	history = core.CloneTurns(history)

	h := &plainHandle{
		provider:          c.Provider,
		tools:             c.Tools,
		history:           history,
		cmds:              initCmds,
		interruptMessages: c.InterruptAppendMessage,
		wakeup:            make(chan struct{}, 1),
		interruptCh:       make(chan struct{}, 1),
	}

	out := make(chan core.ConversationOutput)
	go h.run(ctx, out)
	return h, out, nil
}

type plainHandle struct {
	provider          core.Provider
	tools             core.ToolSetAndRunner
	history           []core.Turn
	interruptMessages []core.Message

	cmdMu       sync.Mutex
	cmds        []core.UserCommand
	wakeup      chan struct{}
	interruptCh chan struct{}
}

func (h *plainHandle) LockCmds()   { h.cmdMu.Lock() }
func (h *plainHandle) UnlockCmds() { h.cmdMu.Unlock() }

func (h *plainHandle) SetCmds(cmds []core.UserCommand) {
	h.cmds = cmds
	select {
	case h.wakeup <- struct{}{}:
	default:
	}
}

func (h *plainHandle) GetCmds() []core.UserCommand {
	out := make([]core.UserCommand, len(h.cmds))
	copy(out, h.cmds)
	return out
}

func (h *plainHandle) InterruptRunningCmd() {
	select {
	case h.interruptCh <- struct{}{}:
	default:
	}
}

func (h *plainHandle) run(ctx context.Context, out chan<- core.ConversationOutput) {
	defer close(out)

	if len(h.history) > 0 {
		actions := make([]core.TurnModifyOrAppendAction, len(h.history))
		for i, t := range h.history {
			actions[i] = core.TurnModifyOrAppendAction{Index: i, Data: t}
		}
		out <- core.ConversationOutput{
			SyncPrimitives: core.SyncPrimitiveModifyOrAppendTurns{Actions: actions},
		}
		out <- core.ConversationOutput{
			AfterEvent: core.KeyNotifyTurnEnd{},
		}
	}

	for {
		h.drainInterrupts(out)

		h.cmdMu.Lock()
		empty := len(h.cmds) == 0
		h.cmdMu.Unlock()

		if empty {
			out <- core.ConversationOutput{
				BeforeEvent: core.KeyNotifyWaitiningPrompt{},
			}
			select {
			case <-h.wakeup:
			case <-h.interruptCh:
				h.emitInterrupt(out)
			case <-ctx.Done():
				out <- core.ConversationOutput{
					BeforeEvent: core.KeyNotifyFailure{Err: ctx.Err()},
				}
				return
			}
			continue
		}

		h.cmdMu.Lock()
		if len(h.cmds) == 0 {
			h.cmdMu.Unlock()
			continue
		}
		cmd := h.cmds[0]
		h.cmds = h.cmds[1:]
		h.cmdMu.Unlock()

		switch c := cmd.(type) {
		case core.EndConversationCommand:
			out <- core.ConversationOutput{
				BeforeEvent: core.KeyNotifyDone{},
			}
			return

		case core.PromptInput:
			p, t := h.provider, h.tools
			if c.Provider != nil {
				p = c.Provider
			}
			msgs := []core.Message{
				core.TextMsg{
					RoleName: "user",
					Content:  string(c.Prompt),
				},
			}
			out <- core.ConversationOutput{
				SyncPrimitives: core.SyncPrimitiveNewTurn{},
			}
			out <- core.ConversationOutput{
				SyncPrimitives: core.SyncPrimitiveAppendMessages{
					Messages: append(msgs,
						core.TrasnparentTextMsg{
							RoleName: "ProviderName",
							Content:  p.Name(),
						},
						core.TrasnparentTextMsg{
							RoleName: "EnvName",
							Content:  t.Name(),
						},
					),
				},
			}

			h.history = append(h.history, msgs)

			turnCtx, cancel := context.WithCancelCause(ctx)
			stream := intra_turn.RunTurn(turnCtx, p, t, h.history,
				intra_turn.DefaultInterruptedToolResult)

			turnDone := false
			for !turnDone {
				select {
				case ev, ok := <-stream:
					if !ok {
						turnDone = true
						break
					}
					out <- ev

				case <-h.interruptCh:
					cancel(errUserInterrupt)
					h.emitInterrupt(out)
				}
			}
			if turnCtx.Err() != nil {
				out <- core.ConversationOutput{
					BeforeEvent: core.KeyNotifyFailure{Err: ctx.Err()},
				}
			}
			cancel(errStageTerminated)
		default:
			continue
		}
	}
}

func (h *plainHandle) drainInterrupts(out chan<- core.ConversationOutput) {
	for {
		select {
		case <-h.interruptCh:
			h.emitInterrupt(out)
		default:
			return
		}
	}
}

func (h *plainHandle) emitInterrupt(out chan<- core.ConversationOutput) {
	if len(h.history) > 0 {
		last := len(h.history) - 1
		h.history[last] = append(h.history[last], h.interruptMessages...)
	}
	out <- core.ConversationOutput{
		BeforeEvent: core.KeyNotifyTurnInterrupted{Err: errUserInterrupt},
		SyncPrimitives: core.SyncPrimitiveAppendMessages{
			Messages: h.interruptMessages,
		},
	}
}
