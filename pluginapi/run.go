package pluginapi

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"time"

	wire "GoCraft/abi/v1/wire"
	"GoCraft/runtime/ipc"
)

const connectTimeout = 10 * time.Second

// Run connects a compiled Go plugin to the GoCraft process and serves it until
// the host requests shutdown. Plugin binaries should call Run from main.
func Run(metadata Metadata, implementation Plugin) error {
	if err := validateMetadata(metadata, implementation); err != nil {
		return err
	}
	options, err := parseRunnerOptions(os.Args[1:])
	if err != nil {
		return err
	}
	stream, err := net.DialTimeout("unix", options.socket, connectTimeout)
	if err != nil {
		return fmt.Errorf("pluginapi: connect to host: %w", err)
	}
	defer stream.Close()
	codec := ipc.NewCodec(stream)

	const sequence = 1
	if err := codec.Send(&wire.Envelope{Seq: sequence, Body: &wire.Envelope_Hello{Hello: &wire.Hello{
		Abi: CurrentVersion, Runtime: "native-go/" + runtime.Version(),
	}}}); err != nil {
		return fmt.Errorf("pluginapi: send handshake: %w", err)
	}
	reply, err := codec.Receive()
	if err != nil {
		return fmt.Errorf("pluginapi: receive handshake: %w", err)
	}
	welcome := reply.GetWelcome()
	if reply.GetSeq() != sequence || welcome == nil {
		return fmt.Errorf("pluginapi: host answered HELLO with %T", reply.GetBody())
	}
	if welcome.GetAbi() != CurrentVersion {
		return fmt.Errorf("pluginapi: host uses ABI %d, plugin uses %d", welcome.GetAbi(), CurrentVersion)
	}
	return serve(codec, newRuntimeState(metadata, implementation))
}
