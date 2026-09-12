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

type serverEventPlugin struct {
	event  string
	handle func(*abi.Event) abi.Verdict
}

func (p serverEventPlugin) Manifest() gcpkg.Manifest {
	return gcpkg.Manifest{ID: "event-test", Subscriptions: []gcpkg.Subscription{{Event: p.event}}}
}
func (p serverEventPlugin) Dispatch(_ context.Context, e *abi.Event) (abi.Verdict, error) {
	return p.handle(e), nil
}
func (serverEventPlugin) Unload(context.Context) error { return nil }

func TestBedrockPlacementCancellationPreservesWorldAndInventory(t *testing.T) {
	for _, item := range []string{"stone", "oak_door", "red_bed", "decorated_pot", "snow", "candle", "oak_slab"} {
		t.Run(item, func(t *testing.T) {
			w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
			defer w.Close()
			p := player.New([16]byte{1}, "builder", player.ClientEditionBedrock)
			p.GameMode = player.GameModeSurvival
			p.Position = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
			p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:" + item, Count: 3}
			g := game.New()
			if err := g.AddPlayer(p); err != nil {
				t.Fatal(err)
			}
			calls := 0
			bus := coreplugin.NewBus(context.Background(), time.Second)
			if err := bus.Attach(serverEventPlugin{event: coreplugin.EventBlockPlace, handle: func(e *abi.Event) abi.Verdict {
				calls++
				return abi.Verdict{Cancelled: true}
			}}); err != nil {
				t.Fatal(err)
			}
			s := &Server{game: g, world: w, sessions: session.NewManager(), plugins: bus}
			pos := spatial.BlockPos{X: 0, Y: 63, Z: 0}
			if item == "snow" || item == "candle" || item == "oak_slab" {
				pos.Y = 64
				w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: item, Properties: map[string]string{"layers": "1", "candles": "1", "type": "bottom"}})
			}
			before := w.GetBlock(0, 64, 0)
			s.applyBedrockBlockInteract(intent.BlockInteractIntent{PlayerUUID: p.UUID, Action: intent.BlockActionUse, Position: pos, Face: 1, HotbarSlot: 0})
			if calls != 1 || p.HeldItem().Count != 3 || !w.GetBlock(0, 64, 0).Equal(before) || !w.GetBlock(0, 65, 0).IsAir() || !w.GetBlock(0, 64, 1).IsAir() {
				t.Fatalf("cancel failed: calls=%d count=%d", calls, p.HeldItem().Count)
			}
		})
	}
}
