package abi

// Internal event and result names used by runtimes until command messages gain
// dedicated protobuf envelope variants.
const (
	EventCommandInvoke    = "gocraft.command.invoke.v1"
	HostCallCommandReply  = "gocraft.command.reply.v1"
	HostCallCommandFailed = "gocraft.command.failed.v1"
)
