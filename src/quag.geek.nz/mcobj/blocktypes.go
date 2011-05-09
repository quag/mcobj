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
			if blockType.empty {
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
	mass         SingularOrAggregate
	transparency Transparency
	empty        bool
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
}

var (
	blockTypeMap map[byte]*BlockType
)
