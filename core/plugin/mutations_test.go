package plugin

import (
	"errors"
	"reflect"
	"testing"

	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
)

func TestMutationQueueDrainsStableFIFO(t *testing.T) {
	queue := NewMutationQueue()
	for _, name := range []string{"first", "second"} {
		if err := queue.Enqueue(abi.HostCall{Type: name}); err != nil {
			t.Fatal(err)
		}
	}
	var applied []string
	count, err := queue.Drain(func(call abi.HostCall) error {
		applied = append(applied, call.Type)
		if call.Type == "first" {
			return queue.Enqueue(abi.HostCall{Type: "next-tick"})
		}
		return nil
	})
	if err != nil || count != 2 {
		t.Fatalf("Drain() = %d, %v", count, err)
	}
	if want := []string{"first", "second"}; !reflect.DeepEqual(applied, want) {
		t.Fatalf("applied = %v, want %v", applied, want)
	}
	count, err = queue.Drain(func(call abi.HostCall) error {
		applied = append(applied, call.Type)
		return nil
	})
	if err != nil || count != 1 || applied[2] != "next-tick" {
		t.Fatalf("next drain = %d, %v, applied %v", count, err, applied)
	}
}

func TestMutationQueueOwnsEnqueuedValues(t *testing.T) {
	queue := NewMutationQueue()
	call := abi.HostCall{Type: "message", Fields: []abi.Value{{
		Kind: abi.ValueList,
		List: []abi.Value{{Kind: abi.ValueBytes, Bytes: []byte("hello")}},
	}}}
	if err := queue.Enqueue(call); err != nil {
		t.Fatal(err)
	}
	call.Fields[0].List[0].Bytes[0] = 'j'
	_, err := queue.Drain(func(queued abi.HostCall) error {
		if got := string(queued.Fields[0].List[0].Bytes); got != "hello" {
			t.Fatalf("queued bytes changed to %q", got)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMutationQueueRejectsCallsAfterClose(t *testing.T) {
	queue := NewMutationQueue()
	queue.Close()
	if err := queue.Enqueue(abi.HostCall{Type: "message"}); !errors.Is(err, ErrMutationQueueClosed) {
		t.Fatalf("Enqueue() error = %v", err)
	}
}
