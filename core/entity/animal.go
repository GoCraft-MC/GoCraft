package entity

import "GoCraft/core/itemregistry"

// Pumpkin/vanilla animal timing constants, expressed in 20 TPS game ticks.
const (
	BabyGrowUpTicks       int32 = 24_000
	LoveDurationTicks     int32 = 600
	BreedingCooldownTicks int32 = 6_000
	BreedingDelayTicks    int32 = 60
)

// AnimalFoodEffect describes the canonical effect of feeding one item to an
// animal. Protocol adapters must not duplicate these tables.
type AnimalFoodEffect struct {
	Accepted       bool
	Breeding       bool
	Taming         bool
	Poisons        bool
	Heal           float32
	GrowthTicks    int32
	TemperIncrease int32
}

// IsAgeableAnimal reports whether the entity uses the generic baby/adult state.
func IsAgeableAnimal(t EntityType) bool {
	switch t {
	case TypeArmadillo, TypeAxolotl, TypeCamel, TypeCat, TypeChicken, TypeCow,
		TypeDonkey, TypeFox, TypeFrog, TypeGoat, TypeHorse, TypeMooshroom,
		TypeMule, TypeOcelot, TypePanda, TypePig, TypeRabbit, TypeSheep,
		TypeSkeletonHorse, TypeSniffer, TypeTurtle, TypeZombieHorse, TypeBee,
		TypeLlama, TypeStrider, TypeTraderLlama, TypeWolf, TypeHoglin:
		return true
	default:
		return false
	}
}

// IsBreedableAnimal reports whether GoCraft has a vanilla food/breeding rule.
// Parrots and mules are deliberately absent because vanilla does not breed them.
func IsBreedableAnimal(t EntityType) bool {
	if !IsAgeableAnimal(t) {
		return false
	}
	switch t {
	case TypeMule, TypeSkeletonHorse, TypeTraderLlama, TypeZombieHorse:
		return false
	default:
		return true
	}
}

func IsTameableAnimal(t EntityType) bool {
	switch t {
	case TypeCat, TypeParrot, TypeWolf, TypeHorse, TypeDonkey, TypeMule,
		TypeLlama, TypeTraderLlama, TypeSkeletonHorse, TypeZombieHorse:
		return true
	default:
		return false
	}
}

// IsAnimalVehicle includes animals which may carry a player. A saddle controls
// most of these but is not required merely to sit on a horse-family mob/camel.
func IsAnimalVehicle(t EntityType) bool {
	switch t {
	case TypeCamel, TypeDonkey, TypeHorse, TypeMule, TypePig, TypeSkeletonHorse,
		TypeStrider, TypeLlama, TypeTraderLlama, TypeZombieHorse:
		return true
	default:
		return false
	}
}

func IsRideableVehicle(t EntityType) bool { return IsBoat(t) || IsMinecart(t) || IsAnimalVehicle(t) }

func VehicleCapacity(t EntityType) int {
	if IsBoat(t) || t == TypeCamel {
		return 2
	}
	if IsAnimalVehicle(t) {
		return 1
	}
	if IsMinecart(t) {
		return 1
	}
	return 0
}

// PassengerIDs returns the occupied seats in vanilla passenger order.
func (e *Entity) PassengerIDs() []int32 {
	if e == nil {
		return nil
	}
	passengers := make([]int32, 0, 2)
	if e.RiderEntityID != 0 {
		passengers = append(passengers, e.RiderEntityID)
	}
	if e.SecondRiderEntityID != 0 {
		passengers = append(passengers, e.SecondRiderEntityID)
	}
	return passengers
}

func (e *Entity) HasPassenger(entityID int32) bool {
	return e != nil && entityID != 0 && (e.RiderEntityID == entityID || e.SecondRiderEntityID == entityID)
}

// AddPassenger occupies the first legal empty seat.
func (e *Entity) AddPassenger(entityID int32) bool {
	if e == nil || entityID == 0 || e.HasPassenger(entityID) {
		return false
	}
	capacity := VehicleCapacity(e.Type)
	if capacity == 0 {
		return false
	}
	if e.RiderEntityID == 0 {
		e.RiderEntityID = entityID
		return true
	}
	if capacity > 1 && e.SecondRiderEntityID == 0 {
		e.SecondRiderEntityID = entityID
		return true
	}
	return false
}

func (e *Entity) RemovePassenger(entityID int32) bool {
	if e == nil || entityID == 0 {
		return false
	}
	if e.RiderEntityID == entityID {
		e.RiderEntityID = e.SecondRiderEntityID
		e.SecondRiderEntityID = 0
		return true
	}
	if e.SecondRiderEntityID == entityID {
		e.SecondRiderEntityID = 0
		return true
	}
	return false
}

func (e *Entity) RemainingBabyTicks() int32 {
	if e == nil || !e.IsBaby {
		return 0
	}
	remaining := BabyGrowUpTicks - e.BabyAgeTicks
	if remaining < 0 {
		return 0
	}
	return remaining
}

// CompatibleBreedingTypes follows vanilla's same-species rule, with the horse
// and donkey cross which produces a mule.
func CompatibleBreedingTypes(a, b EntityType) bool {
	if a == b {
		return IsBreedableAnimal(a)
	}
	return (a == TypeHorse && b == TypeDonkey) || (a == TypeDonkey && b == TypeHorse)
}

func BreedingChildType(a, b EntityType) EntityType {
	if (a == TypeHorse && b == TypeDonkey) || (a == TypeDonkey && b == TypeHorse) {
		return TypeMule
	}
	return a
}

func animalFood(item, animal string) bool {
	return itemregistry.HasTag(item, "minecraft:"+animal+"_food")
}

// FoodEffect is the shared feeding, breeding and taming item table. The tamed
// argument matters for wolves: meat is accepted only by owned wolves.
func FoodEffect(t EntityType, item string, tamed bool) AnimalFoodEffect {
	breed := func(ok bool, heal float32) AnimalFoodEffect {
		return AnimalFoodEffect{Accepted: ok, Breeding: ok, Heal: heal}
	}
	switch t {
	case TypeCow, TypeMooshroom:
		return breed(animalFood(item, "cow"), 0)
	case TypeSheep:
		return breed(animalFood(item, "sheep"), 0)
	case TypeGoat:
		return breed(animalFood(item, "goat"), 0)
	case TypeChicken:
		return breed(animalFood(item, "chicken"), 0)
	case TypePig:
		return breed(animalFood(item, "pig"), 0)
	case TypeRabbit:
		return breed(animalFood(item, "rabbit"), 0)
	case TypeCat:
		ok := animalFood(item, "cat")
		return AnimalFoodEffect{Accepted: ok, Breeding: tamed && ok, Taming: !tamed && ok, Heal: 2}
	case TypeOcelot:
		ok := animalFood(item, "ocelot")
		return AnimalFoodEffect{Accepted: ok, Taming: ok}
	case TypeWolf:
		if !tamed {
			return AnimalFoodEffect{Accepted: item == "minecraft:bone", Taming: item == "minecraft:bone"}
		}
		ok := animalFood(item, "wolf")
		return AnimalFoodEffect{Accepted: ok, Breeding: ok, Heal: 4}
	case TypeParrot:
		if itemregistry.HasTag(item, "minecraft:parrot_poisonous_food") {
			return AnimalFoodEffect{Accepted: true, Poisons: true}
		}
		ok := animalFood(item, "parrot")
		return AnimalFoodEffect{Accepted: ok, Taming: !tamed && ok}
	case TypeHorse, TypeDonkey, TypeMule, TypeSkeletonHorse, TypeZombieHorse:
		switch item {
		case "minecraft:sugar":
			return AnimalFoodEffect{Accepted: true, Heal: 1, GrowthTicks: 600, TemperIncrease: 3}
		case "minecraft:wheat":
			return AnimalFoodEffect{Accepted: true, Heal: 2, GrowthTicks: 400, TemperIncrease: 3}
		case "minecraft:apple":
			return AnimalFoodEffect{Accepted: true, Heal: 3, GrowthTicks: 1_200, TemperIncrease: 3}
		case "minecraft:golden_carrot":
			return AnimalFoodEffect{Accepted: true, Breeding: tamed && t != TypeMule, Heal: 4, GrowthTicks: 1_200, TemperIncrease: 5}
		case "minecraft:golden_apple", "minecraft:enchanted_golden_apple":
			return AnimalFoodEffect{Accepted: true, Breeding: tamed && t != TypeMule, Heal: 10, GrowthTicks: 4_800, TemperIncrease: 10}
		case "minecraft:hay_block":
			return AnimalFoodEffect{Accepted: true, Heal: 20, GrowthTicks: 3_600}
		}
	case TypeLlama, TypeTraderLlama:
		if item == "minecraft:wheat" {
			return AnimalFoodEffect{Accepted: true, Heal: 2, GrowthTicks: 200, TemperIncrease: 3}
		}
		if item == "minecraft:hay_block" {
			return AnimalFoodEffect{Accepted: true, Breeding: tamed && t == TypeLlama, Heal: 10, GrowthTicks: 1_800, TemperIncrease: 6}
		}
	case TypeCamel:
		return breed(animalFood(item, "camel"), 2)
	case TypeTurtle:
		return breed(animalFood(item, "turtle"), 0)
	case TypeFox:
		return breed(animalFood(item, "fox"), 0)
	case TypePanda:
		return breed(animalFood(item, "panda"), 0)
	case TypeBee:
		return breed(animalFood(item, "bee"), 0)
	case TypeHoglin:
		return breed(animalFood(item, "hoglin"), 0)
	case TypeStrider:
		return breed(animalFood(item, "strider"), 0)
	case TypeArmadillo:
		return breed(animalFood(item, "armadillo"), 0)
	case TypeSniffer:
		return breed(animalFood(item, "sniffer"), 0)
	case TypeFrog:
		return breed(animalFood(item, "frog"), 0)
	case TypeAxolotl:
		return breed(animalFood(item, "axolotl"), 0)
	}
	return AnimalFoodEffect{}
}
