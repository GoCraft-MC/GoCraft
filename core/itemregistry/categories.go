package itemregistry

type ToolCategory string

const (
	ToolSword                ToolCategory = "sword"
	ToolAxe                  ToolCategory = "axe"
	ToolPickaxe              ToolCategory = "pickaxe"
	ToolShovel               ToolCategory = "shovel"
	ToolHoe                  ToolCategory = "hoe"
	ToolShears               ToolCategory = "shears"
	ToolFishingRod           ToolCategory = "fishing_rod"
	ToolBrush                ToolCategory = "brush"
	ToolFlintAndSteel        ToolCategory = "flint_and_steel"
	ToolCarrotOnAStick       ToolCategory = "carrot_on_a_stick"
	ToolWarpedFungusOnAStick ToolCategory = "warped_fungus_on_a_stick"
	ToolTrident              ToolCategory = "trident"
	ToolMace                 ToolCategory = "mace"
)

type ToolTier string

const (
	TierWooden    ToolTier = "wooden"
	TierStone     ToolTier = "stone"
	TierIron      ToolTier = "iron"
	TierGolden    ToolTier = "golden"
	TierDiamond   ToolTier = "diamond"
	TierNetherite ToolTier = "netherite"
)

type Rarity string

const (
	RarityCommon   Rarity = "common"
	RarityUncommon Rarity = "uncommon"
	RarityRare     Rarity = "rare"
	RarityEpic     Rarity = "epic"
)
