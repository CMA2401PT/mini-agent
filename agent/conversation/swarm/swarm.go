package swarm

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"mini_agent/core"
)

type TaggedConversationOutput struct {
	ConvID             string
	ConversationOutput core.ConversationOutput
}

type TaggedUserCommand struct {
	ConvID string
	core.UserCommand
}

func (o TaggedConversationOutput) String() string {
	return fmt.Sprintf("%s: %s", o.ConvID, o.ConversationOutput)
}

type SwarmController struct {
	mu       sync.Mutex
	convs    map[string]*conversationInstance
	mergedCh chan TaggedConversationOutput
	nextID   int64
}

type conversationInstance struct {
	id     string
	handle core.ControlHandle
	cancel context.CancelFunc
}

func NewSwarmController() *SwarmController {
	return &SwarmController{
		convs:    make(map[string]*conversationInstance),
		mergedCh: make(chan TaggedConversationOutput, 256),
	}
}

func (s *SwarmController) StartConversation(
	ctx context.Context,
	initCmds []core.UserCommand,
	history []core.Turn,
	template core.ConversationCtrlTemplate) (string, error) {
	convCtx, convCancel := context.WithCancel(ctx)

	handle, ch, err := template.Emit(convCtx, initCmds, history)
	if err != nil {
		convCancel()
		return "", err
	}

	convID := s.genID()
	inst := &conversationInstance{
		id:     convID,
		handle: handle,
		cancel: convCancel,
	}

	s.mu.Lock()
	s.convs[convID] = inst
	s.mu.Unlock()

	go s.fanIn(convID, ch)

	return convID, nil
}

func (s *SwarmController) StopConversation(convID string) error {
	s.mu.Lock()
	inst := s.convs[convID]
	s.mu.Unlock()
	if inst == nil {
		return fmt.Errorf("conversation %s not found", convID)
	}
	inst.handle.InterruptRunningCmd()
	inst.cancel()
	return nil
}

func (s *SwarmController) InterruptConversation(convID string) error {
	s.mu.Lock()
	inst := s.convs[convID]
	s.mu.Unlock()
	if inst == nil {
		return fmt.Errorf("conversation %s not found", convID)
	}
	inst.handle.InterruptRunningCmd()
	return nil
}

func (s *SwarmController) GetInstance(convID string) core.ControlHandle {
	s.mu.Lock()
	inst := s.convs[convID]
	s.mu.Unlock()
	if inst == nil {
		return nil
	}
	return inst.handle
}

func (s *SwarmController) Output() core.OutStream[TaggedConversationOutput] {
	return s.mergedCh
}

func (s *SwarmController) fanIn(convID string, ch core.OutStream[core.ConversationOutput]) {
	for output := range ch {
		s.mergedCh <- TaggedConversationOutput{
			ConvID:             convID,
			ConversationOutput: output,
		}
	}
	s.mu.Lock()
	delete(s.convs, convID)
	s.mu.Unlock()
}

func (s *SwarmController) genID() string {
	s.nextID++
	r := rand.Intn(10000)
	return fmt.Sprintf("conv-%d-%04d", s.nextID, r)
}
