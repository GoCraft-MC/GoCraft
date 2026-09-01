package gocraft

import (
	"fmt"

	abi "GoCraft/abi/v1"
)

func (s *runtimeState) dispatch(incoming *abi.Event) (abi.Verdict, error) {
	if !s.enabled || s.context == nil {
		return abi.Verdict{}, fmt.Errorf("gocraft: plugin is not enabled")
	}
	event, err := eventFrom(incoming)
	if err != nil {
		return abi.Verdict{}, err
	}
	s.context.events.dispatch(event)
	verdict := abi.Verdict{}
	if cancellable, ok := event.(CancellableEvent); ok {
		verdict.Cancelled = cancellable.Cancelled()
	}
	return verdict, nil
}
