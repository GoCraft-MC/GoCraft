package server

import (
	"context"
	"testing"
	"time"

	"GoCraft/core/game"
	"GoCraft/core/intent"
	"GoCraft/core/player"
	coreplugin "GoCraft/core/plugin"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
	"github.com/GoCraft-MC/gocraft-abi/gcpkg"
)

type serverCancellingPlugin struct{}

func (serverCancellingPlugin) Manifest() gcpkg.Manifest {
	return gcpkg.Manifest{ID: "protect", Subscriptions: []gcpkg.Subscription{{Event: coreplugin.EventBlockBreak}}}
}
func (serverCancellingPlugin) Dispatch(context.Context, *abi.Event) (abi.Verdict, error) {
	return abi.Verdict{Cancelled: true}, nil
}
func (serverCancellingPlugin) Unload(context.Context) error { return nil }

func TestBedrockPluginCanCancelBlockBreak(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	block := coreworld.Block{Namespace: "minecraft", Name: "stone"}
	w.SetBlock(1, 64, 0, block)
	p := player.New([16]byte{2}, "builder", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Position = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:iron_pickaxe", Count: 1}
	g := game.New()
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	plugins := coreplugin.NewBus(context.Background(), time.Second)
	if err := plugins.Attach(serverCancellingPlugin{}); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g, world: w, sessions: session.NewManager(), plugins: plugins}
	s.applyBedrockBlockInteract(intent.BlockInteractIntent{
		PlayerUUID: p.UUID, Action: intent.BlockActionBreak,
		Position: spatial.BlockPos{X: 1, Y: 64, Z: 0}, HotbarSlot: 0,
	})
	if got := w.GetBlock(1, 64, 0); !got.Equal(block) {
		t.Fatalf("cancelled block became %q", got.ResourceLocation())
	}
	if got := p.Inventory[player.HotbarStart].Damage; got != 0 {
		t.Fatalf("cancelled break damaged tool by %d", got)
	}
}
