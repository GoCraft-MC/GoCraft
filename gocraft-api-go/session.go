package gocraft

import (
	"fmt"
	"io"

	wire "GoCraft/abi/v1/wire"
	"GoCraft/runtime/ipc"
)

func serve(codec *ipc.Codec, state *runtimeState) error {
	session := &pluginSession{codec: codec, state: state}
	work := make(chan *wire.Envelope, 8)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		defer state.disable()
		for envelope := range work {
			session.handle(envelope)
		}
	}()
	stop := func() {
		close(work)
		<-stopped
	}
	defer codec.Close()

	for {
		envelope, err := codec.Receive()
		if err != nil {
			stop()
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("gocraft: receive host message: %w", err)
		}
		switch envelope.GetBody().(type) {
		case *wire.Envelope_Ping:
			if err := codec.Send(&wire.Envelope{Seq: envelope.GetSeq(),
				Body: &wire.Envelope_Pong{Pong: &wire.Pong{}}}); err != nil {
				stop()
				return err
			}
		case *wire.Envelope_Ready:
			// The host is now accepting players; no plugin callback is needed.
		case *wire.Envelope_Shutdown:
			stop()
			return nil
		case *wire.Envelope_Load, *wire.Envelope_Dispatch, *wire.Envelope_Unload,
			*wire.Envelope_Invoke:
			work <- envelope
		default:
			stop()
			return fmt.Errorf("gocraft: unexpected host message %T", envelope.GetBody())
		}
	}
}
