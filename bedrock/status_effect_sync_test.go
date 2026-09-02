package bedrock

import (
	"testing"

	"GoCraft/core/player"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestStatusEffectOperation(t *testing.T) {
	before := player.StatusEffect{ID: "minecraft:poison", Duration: 100, ShowParticles: true}
	if got := statusEffectOperation(player.StatusEffect{}, false, before); got != packet.MobEffectAdd {
		t.Fatalf("new effect operation = %d", got)
	}
	current := before
	current.Duration--
	if got := statusEffectOperation(before, true, current); got != 0 {
		t.Fatalf("normal tick operation = %d", got)
	}
	current.Duration = 200
	if got := statusEffectOperation(before, true, current); got != packet.MobEffectModify {
		t.Fatalf("refreshed effect operation = %d", got)
	}
}
