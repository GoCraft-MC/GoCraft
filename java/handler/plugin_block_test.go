package handler

import (
	"context"
	"testing"
	"time"

	"GoCraft/core/player"
	coreplugin "GoCraft/core/plugin"
	coreworld "GoCraft/core/world"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
)

type cancellingPlugin struct{}

func (cancellingPlugin) Manifest() coreplugin.Manifest {
	return coreplugin.Manifest{ID: "protect", Subscriptions: []coreplugin.Subscription{{Event: coreplugin.EventBlockBreak}}}
}
func (cancellingPlugin) Dispatch(context.Context, *abi.Event) (abi.Verdict, error) {
	return abi.Verdict{Cancelled: true}, nil
}
func (cancellingPlugin) Unload(context.Context) error { return nil }

func TestJavaPluginCanCancelBlockBreak(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	block := coreworld.Block{Namespace: "minecraft", Name: "stone"}
	w.SetBlock(1, 64, 0, block)
	p := player.New([16]byte{1}, "builder", player.ClientEditionJava)
	p.GameMode = player.GameModeCreative
	plugins := coreplugin.NewBus(context.Background(), time.Second)
	if err := plugins.Attach(cancellingPlugin{}); err != nil {
		t.Fatal(err)
	}
	pkt := protocol.NewBuilder(packetIDPlayerAction).
		VarInt(actionStatusStartDigging).
		Long(packBlockPos(1, 64, 0)).
		Byte(1).
		VarInt(1).
		Build()
	err := handlePlayerActionWithContext(pkt, p, w, session.NewManager(), nil, func() int32 { return 1 }, plugins)
	if err != nil {
		t.Fatal(err)
	}
	if got := w.GetBlock(1, 64, 0); !got.Equal(block) {
		t.Fatalf("cancelled block became %q", got.ResourceLocation())
	}
}
