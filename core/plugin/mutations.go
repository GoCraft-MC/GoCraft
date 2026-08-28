package plugin

import (
	"errors"
	"fmt"
	"sync"

	abi "GoCraft/abi/v1"
)

var ErrMutationQueueClosed = errors.New("plugin mutation queue is closed")

// MutationQueue buffers plugin host calls until the simulation tick drains it.
type MutationQueue struct {
	mu     sync.Mutex
	calls  []abi.HostCall
	closed bool
}

func NewMutationQueue() *MutationQueue {
	return &MutationQueue{}
}

// Enqueue satisfies Host. It never applies the call on the runtime goroutine.
func (q *MutationQueue) Enqueue(call abi.HostCall) error {
	if call.Type == "" {
		return fmt.Errorf("plugin host call: empty type")
	}
	call = cloneHostCall(call)
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrMutationQueueClosed
	}
	q.calls = append(q.calls, call)
	return nil
}

// Drain applies the calls present at its start in FIFO order.
func (q *MutationQueue) Drain(apply func(abi.HostCall) error) (int, error) {
	q.mu.Lock()
	calls := q.calls
	q.calls = nil
	q.mu.Unlock()
	var joined error
	for _, call := range calls {
		if err := apply(call); err != nil {
			joined = errors.Join(joined, err)
		}
	}
	return len(calls), joined
}

func (q *MutationQueue) Close() {
	q.mu.Lock()
	q.closed = true
	q.mu.Unlock()
}

func cloneHostCall(call abi.HostCall) abi.HostCall {
	copyCall := abi.HostCall{Type: call.Type, Fields: make([]abi.Value, len(call.Fields))}
	for index, value := range call.Fields {
		copyCall.Fields[index] = cloneValue(value)
	}
	return copyCall
}

func cloneValue(value abi.Value) abi.Value {
	value.Bytes = append([]byte(nil), value.Bytes...)
	if value.List != nil {
		value.List = make([]abi.Value, len(value.List))
		for index, child := range value.List {
			value.List[index] = cloneValue(child)
		}
	}
	return value
}
