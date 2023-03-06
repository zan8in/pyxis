package pyxis

import "sync"

type State int

const (
	Init State = iota
	Scan
	Done
	Guard
)

type Phase struct {
	sync.RWMutex
	State
}

func (phase *Phase) Is(state State) bool {
	phase.RLock()
	defer phase.RUnlock()

	return phase.State == state
}

func (phase *Phase) Set(state State) {
	phase.Lock()
	defer phase.Unlock()

	phase.State = state
}
