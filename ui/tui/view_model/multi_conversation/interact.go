package multi_conversation

// type InteractType int

// const (
// 	InteractInput InteractType = iota
// 	InteractQuit
// 	InteractInterrupt
// )

// type InteractEvent struct {
// 	ConvID string
// 	Type   InteractType
// 	Prompt string
// }

// func mapInteract(ev agent_interact.UserInteract) InteractEvent {
// 	switch e := ev.(type) {
// 	case agent_interact.UserInput:
// 		return InteractEvent{Type: InteractInput, Prompt: e.Prompt}
// 	case agent_interact.UserInterrupt:
// 		return InteractEvent{Type: InteractInterrupt}
// 	case agent_interact.UserQuit:
// 		return InteractEvent{Type: InteractQuit}
// 	default:
// 		return InteractEvent{Type: InteractQuit}
// 	}
// }
