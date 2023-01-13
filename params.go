package template

var (
	block_store_name   = "_blocks_"
	block_remains_name = "__parent__"
)

type Params map[string]any

func cop(p Params) Params {
	np := make(Params)
	for k, v := range p {
		np[k] = v
	}

	return np
}

func (p Params) getBlock(name string) *BlockDirect {
	if _blocks, ok := p[block_store_name]; ok {
		_blocksMap := _blocks.(map[string]*BlockDirect)
		if block, ok := _blocksMap[name]; ok {
			return block
		}
	}

	return nil
}

func (p Params) setBlock(name string, block *BlockDirect) {
	if block == nil {
		return
	}
	var blocks map[string]*BlockDirect
	if _, ok := p[block_store_name]; ok {
		blocks = p[block_store_name].(map[string]*BlockDirect)
	} else {
		blocks = make(map[string]*BlockDirect)
	}
	blocks[name] = block
	p[block_store_name] = blocks
}

func (p Params) setBlockRemains(remains string) {
	p[block_remains_name] = remains
}
