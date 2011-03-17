package main

type BoundaryLocator struct {
	describer BlockDescriber
}

func (b *BoundaryLocator) Init() {
	var describer = new(Describer)
	describer.Init()
	b.describer = describer
}

func (b *BoundaryLocator) IsBoundary(blockId, otherBlockId uint16) bool {
	var (
		block = b.describer.BlockInfo(byte(blockId & 0xff))
		other = b.describer.BlockInfo(byte(otherBlockId & 0xff))
	)

	if !block.IsEmpty() {
		if other.IsEmpty() {
			return true
		}

		if block.IsItem() {
			return true
		}

		if other.IsTransparent() && (other.IsItem() || blockId&0xff != otherBlockId&0xff) {
			return true
		}
	}
	return false
}

type Describer struct {
	unknown BlockInfo
	cache   map[byte]BlockInfoByte
}

func (d *Describer) Init() {
	d.cache = make(map[byte]BlockInfoByte)
	for i := 0; i < 256; i++ {
		var blockId = byte(i)
		var blockType, hasType = blockTypeMap[blockId]
		var value byte
		if hasType {
			if blockType.mass == Mass {
				value += 1
			}
			if blockType.transparency == Transparent {
				value += 2
			}
			if blockType.blockId == 0 || (hideStone && blockType.blockId == 1) {
				value += 4
			}
		} else {
			value = 1
		}
		var infoByte = BlockInfoByte(value)
		d.cache[blockId] = infoByte
	}
}

func (d *Describer) BlockInfo(blockId byte) BlockInfo {
	var info, _ = d.cache[blockId]
	return info
}

type BlockDescriber interface {
	BlockInfo(blockId byte) BlockInfo
}

type BlockInfo interface {
	IsItem() bool
	IsMass() bool
	IsOpaque() bool
	IsTransparent() bool
	IsEmpty() bool
}

type BlockType struct {
	blockId      byte
	name         string
	mass         SingularOrAggregate
	transparency Transparency
}

type Transparency bool
type SingularOrAggregate bool

const (
	Transparent Transparency = true
	Opaque      Transparency = false

	Item SingularOrAggregate = true
	Mass SingularOrAggregate = false
)

type BlockInfoByte byte

func (b BlockInfoByte) IsItem() bool {
	return b&1 == 0
}

func (b BlockInfoByte) IsMass() bool {
	return b&1 != 0
}

func (b BlockInfoByte) IsOpaque() bool {
	return b&2 == 0
}

func (b BlockInfoByte) IsTransparent() bool {
	return b&2 != 0
}

func (b BlockInfoByte) IsEmpty() bool {
	return b&4 != 0
}

func init() {
	blockTypeMap = make(map[byte]*BlockType)
	for _, blockType := range blockTypesList {
		blockTypeMap[blockType.blockId] = blockType
	}
}

var (
	blockTypeMap map[byte]*BlockType

	blockTypesList = []*BlockType{
		&BlockType{0, "Air", Mass, Transparent},
		&BlockType{1, "Stone", Mass, Opaque},
		&BlockType{2, "Grass", Mass, Opaque},
		&BlockType{3, "Dirt", Mass, Opaque},
		&BlockType{4, "Cobblestone", Mass, Opaque},
		&BlockType{5, "Wooden Plank", Mass, Opaque},
		&BlockType{6, "Sapling", Item, Transparent},
		&BlockType{7, "Bedrock", Mass, Opaque},
		&BlockType{8, "Water", Mass, Transparent},
		&BlockType{9, "Stationary water", Mass, Transparent},
		&BlockType{10, "Lava", Mass, Transparent},
		&BlockType{11, "Stationary lava", Mass, Transparent},
		&BlockType{12, "Sand", Mass, Opaque},
		&BlockType{13, "Gravel", Mass, Opaque},
		&BlockType{14, "Gold ore", Mass, Opaque},
		&BlockType{15, "Iron ore", Mass, Opaque},
		&BlockType{16, "Coal ore", Mass, Opaque},
		&BlockType{17, "Wood", Mass, Opaque},
		&BlockType{18, "Leaves", Mass, Transparent},
		&BlockType{19, "Sponge", Item, Transparent},
		&BlockType{20, "Glass", Mass, Transparent},
		&BlockType{21, "Lapis Lazuli Ore", Mass, Opaque},
		&BlockType{22, "Lapis Lazuli Block", Mass, Opaque},
		&BlockType{23, "Dispenser", Mass, Opaque},
		&BlockType{24, "Sandstone", Mass, Opaque},
		&BlockType{25, "Note Block", Mass, Opaque},
		&BlockType{26, "Bed", Item, Transparent},
		&BlockType{35, "Wool", Mass, Opaque},
		&BlockType{37, "Yellow flower", Item, Transparent},
		&BlockType{38, "Red rose", Item, Transparent},
		&BlockType{39, "Brown Mushroom", Item, Transparent},
		&BlockType{40, "Red Mushroom", Item, Transparent},
		&BlockType{41, "Gold Block", Mass, Opaque},
		&BlockType{42, "Iron Block", Mass, Opaque},
		&BlockType{43, "Double Stone Slab", Mass, Opaque},
		&BlockType{44, "Stone Slab", Item, Transparent},
		&BlockType{45, "Brick", Mass, Opaque},
		&BlockType{46, "TNT", Mass, Opaque},
		&BlockType{47, "Bookshelf", Mass, Opaque},
		&BlockType{48, "Moss Stone", Mass, Opaque},
		&BlockType{49, "Obsidian", Mass, Opaque},
		&BlockType{50, "Torch", Item, Transparent},
		&BlockType{51, "Fire", Item, Transparent},
		&BlockType{52, "Monster Spawner", Item, Transparent},
		&BlockType{53, "Wooden Stairs", Item, Transparent},
		&BlockType{54, "Chest", Mass, Opaque},
		&BlockType{55, "Redstone Wire", Item, Transparent},
		&BlockType{56, "Diamond Ore", Mass, Opaque},
		&BlockType{57, "Diamond Block", Mass, Opaque},
		&BlockType{58, "Workbench", Mass, Opaque},
		&BlockType{59, "Crops", Item, Transparent},
		&BlockType{60, "Soil", Mass, Opaque},
		&BlockType{61, "Furnace", Mass, Opaque},
		&BlockType{62, "Burning Furnace", Mass, Opaque},
		&BlockType{63, "Sign Post", Item, Transparent},
		&BlockType{64, "Wooden Door", Item, Transparent},
		&BlockType{65, "Ladder", Item, Transparent},
		&BlockType{66, "Minecart Tracks", Item, Transparent},
		&BlockType{67, "Cobblestone Stairs", Item, Transparent},
		&BlockType{68, "Wall Sign", Item, Transparent},
		&BlockType{69, "Lever", Item, Transparent},
		&BlockType{70, "Stone Pressure Plate", Item, Transparent},
		&BlockType{71, "Iron Door", Item, Transparent},
		&BlockType{72, "Wooden Pressure Plate", Item, Transparent},
		&BlockType{73, "Redstone Ore", Mass, Opaque},
		&BlockType{74, "Glowing Redstone Ore", Mass, Opaque},
		&BlockType{75, "Redstone torch (\"off\" state)", Item, Transparent},
		&BlockType{76, "Redstone torch (\"on\" state)", Item, Transparent},
		&BlockType{77, "Stone Button", Item, Transparent},
		&BlockType{78, "Snow", Item, Transparent},
		&BlockType{79, "Ice", Mass, Transparent},
		&BlockType{80, "Snow Block", Mass, Opaque},
		&BlockType{81, "Cactus", Item, Transparent},
		&BlockType{82, "Clay", Mass, Opaque},
		&BlockType{83, "Sugar Cane", Item, Transparent},
		&BlockType{84, "Jukebox", Mass, Opaque},
		&BlockType{85, "Fence", Item, Transparent},
		&BlockType{86, "Pumpkin", Mass, Opaque},
		&BlockType{87, "Netherrack", Mass, Opaque},
		&BlockType{88, "Soul Sand", Mass, Opaque},
		&BlockType{89, "Glowstone", Mass, Opaque},
		&BlockType{90, "Portal", Mass, Opaque},
		&BlockType{91, "Jack-O-Lantern", Mass, Opaque},
		&BlockType{92, "Cake Block", Item, Transparent},
		&BlockType{93, "Redstone Repeater (\"off\" state)", Item, Transparent},
		&BlockType{94, "Redstone Repeater (\"on\" state)", Item, Transparent},
	}
)
