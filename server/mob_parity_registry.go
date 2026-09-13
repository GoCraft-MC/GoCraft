package server

import corentity "GoCraft/core/entity"

// mobParitySpec is the exhaustive Java 1.21.4 living-mob parity contract.
//
// This table is deliberately independent from spawn tables. A mob being
// registered, spawnable, or renderable does not mean its behaviour is parity
// complete. Every row must have its controller, navigation, targeting,
// interactions, attacks, environment rules and lifecycle validated before the
// parity project is considered complete.
type mobParitySpec struct {
	Type       corentity.EntityType
	Controller string
	Navigation string
	Temper     string
}

var minecraft1214MobParitySpecs = [...]mobParitySpec{
	{corentity.TypeAllay, "allay-brain", "flying", "passive"},
	{corentity.TypeArmadillo, "armadillo-brain", "ground", "passive"},
	{corentity.TypeAxolotl, "axolotl-brain", "amphibious", "passive"},
	{corentity.TypeBat, "bat-goals", "flying", "passive"},
	{corentity.TypeCamel, "camel-brain", "ground", "passive"},
	{corentity.TypeCat, "cat-goals", "ground", "passive"},
	{corentity.TypeChicken, "animal-goals", "ground", "passive"},
	{corentity.TypeCod, "schooling-fish", "aquatic", "passive"},
	{corentity.TypeCow, "animal-goals", "ground", "passive"},
	{corentity.TypeDonkey, "horse-goals", "ground", "passive"},
	{corentity.TypeFox, "fox-goals", "ground", "passive"},
	{corentity.TypeFrog, "frog-brain", "amphibious", "passive"},
	{corentity.TypeGlowSquid, "squid-goals", "aquatic", "passive"},
	{corentity.TypeGoat, "goat-brain", "ground", "neutral"},
	{corentity.TypeHorse, "horse-goals", "ground", "passive"},
	{corentity.TypeMooshroom, "animal-goals", "ground", "passive"},
	{corentity.TypeMule, "horse-goals", "ground", "passive"},
	{corentity.TypeOcelot, "ocelot-goals", "ground", "passive"},
	{corentity.TypePanda, "panda-goals", "ground", "neutral"},
	{corentity.TypeParrot, "parrot-goals", "flying", "passive"},
	{corentity.TypePig, "animal-goals", "ground", "passive"},
	{corentity.TypePufferfish, "pufferfish-goals", "aquatic", "neutral"},
	{corentity.TypeRabbit, "rabbit-goals", "ground", "passive"},
	{corentity.TypeSalmon, "schooling-fish", "aquatic", "passive"},
	{corentity.TypeSheep, "sheep-goals", "ground", "passive"},
	{corentity.TypeSkeletonHorse, "horse-goals", "ground", "passive"},
	{corentity.TypeSniffer, "sniffer-brain", "ground", "passive"},
	{corentity.TypeSquid, "squid-goals", "aquatic", "passive"},
	{corentity.TypeTadpole, "tadpole-goals", "aquatic", "passive"},
	{corentity.TypeTropicalFish, "schooling-fish", "aquatic", "passive"},
	{corentity.TypeTurtle, "turtle-goals", "amphibious", "passive"},
	{corentity.TypeVillager, "villager-brain", "ground", "passive"},
	{corentity.TypeWanderingTrader, "wandering-trader-goals", "ground", "passive"},
	{corentity.TypeZombieHorse, "horse-goals", "ground", "passive"},

	{corentity.TypeBee, "bee-goals", "flying", "neutral"},
	{corentity.TypeDolphin, "dolphin-goals", "aquatic", "neutral"},
	{corentity.TypeIronGolem, "iron-golem-goals", "ground", "neutral"},
	{corentity.TypeLlama, "llama-goals", "ground", "neutral"},
	{corentity.TypePolarBear, "polar-bear-goals", "ground", "neutral"},
	{corentity.TypeSnowGolem, "snow-golem-goals", "ground", "neutral"},
	{corentity.TypeStrider, "strider-goals", "lava", "passive"},
	{corentity.TypeTraderLlama, "llama-goals", "ground", "neutral"},
	{corentity.TypeWolf, "wolf-goals", "ground", "neutral"},
	{corentity.TypeZombifiedPiglin, "zombified-piglin-goals", "ground", "neutral"},

	{corentity.TypeBlaze, "blaze-goals", "flying", "hostile"},
	{corentity.TypeBogged, "skeleton-goals", "ground", "hostile"},
	{corentity.TypeBreeze, "breeze-brain", "jumping", "hostile"},
	{corentity.TypeCaveSpider, "spider-goals", "climbing", "hostile"},
	{corentity.TypeCreaker, "creaking-brain", "ground", "hostile"},
	{corentity.TypeCreeper, "creeper-goals", "ground", "hostile"},
	{corentity.TypeDrowned, "drowned-goals", "amphibious", "hostile"},
	{corentity.TypeElderGuardian, "guardian-goals", "aquatic", "hostile"},
	{corentity.TypeEnderman, "enderman-goals", "teleporting", "neutral"},
	{corentity.TypeEndermite, "endermite-goals", "ground", "hostile"},
	{corentity.TypeEnderDragon, "ender-dragon-phases", "boss-flight", "boss"},
	{corentity.TypeEvoker, "evoker-goals", "ground", "hostile"},
	{corentity.TypeGhast, "ghast-goals", "flying", "hostile"},
	{corentity.TypeGiant, "giant-goals", "ground", "hostile"},
	{corentity.TypeGuardian, "guardian-goals", "aquatic", "hostile"},
	{corentity.TypeHoglin, "hoglin-brain", "ground", "neutral"},
	{corentity.TypeHusk, "zombie-goals", "ground", "hostile"},
	{corentity.TypeIllusioner, "illusioner-goals", "ground", "hostile"},
	{corentity.TypeMagmaCube, "magma-cube-goals", "jumping", "hostile"},
	{corentity.TypePhantom, "phantom-goals", "flying", "hostile"},
	{corentity.TypePiglin, "piglin-brain", "ground", "neutral"},
	{corentity.TypePiglinBrute, "piglin-brute-brain", "ground", "hostile"},
	{corentity.TypePillager, "pillager-goals", "ground", "hostile"},
	{corentity.TypeRavager, "ravager-goals", "ground", "hostile"},
	{corentity.TypeShulker, "shulker-goals", "anchored", "hostile"},
	{corentity.TypeSilverfish, "silverfish-goals", "ground", "hostile"},
	{corentity.TypeSkeleton, "skeleton-goals", "ground", "hostile"},
	{corentity.TypeSlime, "slime-goals", "jumping", "hostile"},
	{corentity.TypeSpider, "spider-goals", "climbing", "neutral"},
	{corentity.TypeStray, "skeleton-goals", "ground", "hostile"},
	{corentity.TypeVex, "vex-goals", "flying", "hostile"},
	{corentity.TypeVindicator, "vindicator-goals", "ground", "hostile"},
	{corentity.TypeWarden, "warden-brain", "ground", "hostile"},
	{corentity.TypeWitch, "witch-goals", "ground", "hostile"},
	{corentity.TypeWither, "wither-boss", "boss-flight", "boss"},
	{corentity.TypeWitherSkeleton, "wither-skeleton-goals", "ground", "hostile"},
	{corentity.TypeZoglin, "zoglin-brain", "ground", "hostile"},
	{corentity.TypeZombie, "zombie-goals", "ground", "hostile"},
	{corentity.TypeZombieVillager, "zombie-goals", "ground", "hostile"},
}
