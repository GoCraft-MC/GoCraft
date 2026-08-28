package server

import (
	"math"
	"strings"

	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/handler"
)

const furnaceSlotCount = 3

type furnaceState struct {
	BurnTime     int
	BurnDuration int
	CookTime     int
	CookDuration int
	RecipesUsed  map[string]furnaceRecipeUse
}

type furnaceRecipeUse struct {
	Count      int
	Experience float32
}

func (state *furnaceState) recordRecipe(recipe handler.CookingRecipeDescription) {
	if state.RecipesUsed == nil {
		state.RecipesUsed = make(map[string]furnaceRecipeUse)
	}
	used := state.RecipesUsed[recipe.Name]
	used.Count++
	used.Experience = recipe.Experience
	state.RecipesUsed[recipe.Name] = used
}

func (state *furnaceState) extractExperience() int32 {
	var total float32
	for _, used := range state.RecipesUsed {
		total += used.Experience * float32(used.Count)
	}
	clear(state.RecipesUsed)
	return int32(math.Floor(float64(total)))
}

type furnaceKey struct {
	Dimension int32
	Position  spatial.BlockPos
}

func loadFurnaceSlots(w *coreworld.World, pos spatial.BlockPos) []player.ItemStack {
	slots := make([]player.ItemStack, furnaceSlotCount)
	for _, item := range w.ContainerItems(int(pos.X), int(pos.Y), int(pos.Z)) {
		if item.Slot >= 0 && item.Slot < furnaceSlotCount && item.ItemID != "" && item.Count > 0 {
			slots[item.Slot] = player.ItemStack{ItemID: item.ItemID, Count: item.Count, Damage: item.Damage, Enchantments: item.Enchantments}
		}
	}
	return slots
}

func persistFurnaceSlots(w *coreworld.World, pos spatial.BlockPos, blockID string, slots []player.ItemStack) {
	items := make([]coreworld.ContainerItem, 0, furnaceSlotCount)
	for slot := 0; slot < furnaceSlotCount && slot < len(slots); slot++ {
		stack := slots[slot]
		if !stack.IsEmpty() {
			items = append(items, coreworld.ContainerItem{Slot: slot, ItemID: stack.ItemID, Count: stack.Count, Damage: stack.Damage, Enchantments: stack.Enchantments})
		}
	}
	w.SetContainerItems(int(pos.X), int(pos.Y), int(pos.Z), blockID, items)
}

func (s *Server) openBedrockFurnace(p *player.Player, pos spatial.BlockPos, blockID string) {
	p.OpenContainerID = 1
	p.OpenContainerKind = blockID
	p.OpenContainerPos = pos
	p.OpenContainerPartnerPos = spatial.BlockPos{}
	p.OpenContainerHasPartner = false
	p.ContainerSlots = loadFurnaceSlots(s.worldForPlayer(p), pos)
	s.furnaceStateForDimension(p.Dimension, pos)
}

func (s *Server) furnaceStateFor(pos spatial.BlockPos) *furnaceState {
	return s.furnaceStateForDimension(dimensionOverworld, pos)
}

func (s *Server) furnaceStateForDimension(dimension int32, pos spatial.BlockPos) *furnaceState {
	if s.furnaces == nil {
		s.furnaces = make(map[furnaceKey]*furnaceState)
	}
	key := furnaceKey{Dimension: dimension, Position: pos}
	state := s.furnaces[key]
	if state == nil {
		state = &furnaceState{}
		s.furnaces[key] = state
	}
	return state
}

func furnaceCanAccept(slots []player.ItemStack, result player.ItemStack) bool {
	if len(slots) < furnaceSlotCount || result.IsEmpty() {
		return false
	}
	output := slots[2]
	if output.IsEmpty() {
		return result.Count <= player.MaxStackSize(result.ItemID)
	}
	return output.ItemID == result.ItemID && output.Damage == result.Damage &&
		output.Count+result.Count <= player.MaxStackSize(output.ItemID)
}

func consumeFurnaceFuel(slots []player.ItemStack) {
	if len(slots) < furnaceSlotCount || slots[1].IsEmpty() {
		return
	}
	remainder := handler.FurnaceFuelRemainder(slots[1].ItemID)
	slots[1].Count--
	if slots[1].Count <= 0 {
		slots[1] = remainder
	}
}

// tickFurnaces follows Pumpkin's furnace-like block entity flow: burn fuel,
// progress one matching recipe, craft into the output slot, and decay stalled
// progress. Cooking recipes and durations remain Java 1.21.4 authoritative.
func (s *Server) tickFurnaces() {
	// Opening a Java furnace is handled by the Java adapter, so discover it here
	// and attach a simulation state on the next canonical tick.
	s.game.OnlinePlayers(func(p *player.Player) {
		if handler.IsFurnaceContainer(p.OpenContainerKind) {
			s.furnaceStateForDimension(p.Dimension, p.OpenContainerPos)
		}
	})

	for key, state := range s.furnaces {
		pos := key.Position
		dimensionWorld := s.worldForDimension(key.Dimension)
		block := dimensionWorld.GetBlock(int(pos.X), int(pos.Y), int(pos.Z))
		if !handler.IsFurnaceContainer(block.ResourceLocation()) {
			delete(s.furnaces, key)
			continue
		}
		station := block.ResourceLocation()
		slots := loadFurnaceSlots(dimensionWorld, pos)
		before := [furnaceSlotCount]player.ItemStack{}
		copy(before[:], slots)

		if state.BurnTime > 0 {
			state.BurnTime--
		}
		recipe, hasRecipe := handler.FindCookingRecipe(station, slots[0].ItemID)
		canCook := hasRecipe && furnaceCanAccept(slots, recipe.Result)
		if state.BurnTime == 0 && canCook && len(slots) > 1 && handler.FurnaceFuelDuration(slots[1].ItemID) > 0 {
			duration := handler.FurnaceFuelDuration(slots[1].ItemID)
			name := strings.TrimPrefix(station, "minecraft:")
			if name == "blast_furnace" || name == "lit_blast_furnace" || name == "smoker" || name == "lit_smoker" {
				duration /= 2
			}
			state.BurnTime = duration
			state.BurnDuration = duration
			consumeFurnaceFuel(slots)
		}

		if state.BurnTime > 0 && canCook {
			state.CookDuration = recipe.CookingTime
			if state.CookDuration <= 0 {
				state.CookDuration = 200
			}
			state.CookTime++
			if state.CookTime >= state.CookDuration {
				if slots[0].ItemID == "minecraft:wet_sponge" && slots[1].ItemID == "minecraft:bucket" {
					slots[1] = player.ItemStack{ItemID: "minecraft:water_bucket", Count: 1}
				}
				slots[0].Count--
				if slots[0].Count <= 0 {
					slots[0] = player.ItemStack{}
				}
				if slots[2].IsEmpty() {
					slots[2] = recipe.Result
				} else {
					slots[2].Count += recipe.Result.Count
				}
				state.recordRecipe(recipe)
				state.CookTime = 0
			}
		} else if state.CookTime > 0 {
			state.CookTime -= 2
			if state.CookTime < 0 {
				state.CookTime = 0
			}
		}

		after := [furnaceSlotCount]player.ItemStack{}
		copy(after[:], slots)
		slotsChanged := before != after
		if slotsChanged {
			persistFurnaceSlots(dimensionWorld, pos, station, slots)
		}

		lit := state.BurnTime > 0
		if (block.Properties["lit"] == "true") != lit {
			updated := block
			updated.Properties = make(map[string]string, len(block.Properties)+1)
			for key, value := range block.Properties {
				updated.Properties[key] = value
			}
			if lit {
				updated.Properties["lit"] = "true"
			} else {
				updated.Properties["lit"] = "false"
			}
			dimensionWorld.SetBlock(int(pos.X), int(pos.Y), int(pos.Z), updated)
			if key.Dimension == dimensionOverworld {
				handler.BroadcastBlockChange(coreworld.BlockChange{X: int(pos.X), Y: int(pos.Y), Z: int(pos.Z), Block: updated}, s.sessions)
			}
		}

		s.game.OnlinePlayers(func(p *player.Player) {
			if p.Dimension != key.Dimension || !handler.IsFurnaceContainer(p.OpenContainerKind) || p.OpenContainerPos != pos {
				return
			}
			if slotsChanged {
				p.ContainerStateID++
			}
			p.ContainerSlots = append(p.ContainerSlots[:0], slots...)
			if p.Edition == player.ClientEditionBedrock && s.bedrockListener != nil {
				s.bedrockListener.SyncFurnaceContainer(p, state.CookTime, state.BurnTime, state.BurnDuration, state.CookDuration)
			}
			if p.Edition == player.ClientEditionJava {
				if sess, ok := s.sessions.Get(p.UUID); ok {
					_ = handler.SyncFurnaceContainer(sess.Conn, p, state.CookTime, state.BurnTime, state.BurnDuration, state.CookDuration)
				}
			}
		})
	}
}
