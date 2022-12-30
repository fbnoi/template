package template

var (
	block_store_name = "__blocks__"
)

type Params map[string]any

func copyParams(ps Params) Params {
	nps := make(map[string]any)
	for n, v := range ps {
		nps[n] = v
	}

	return nps
}

func (p Params) getBlock(name string) *BlockDirect {
	if blockIfs, ok := p[block_store_name]; ok {
		blocks := blockIfs.(map[string]*BlockDirect)
		if block, ok := blocks[name]; ok {
			return block
		}
	}

	return nil
}

func (p Params) setBlock(name string, block *BlockDirect) {
	if block == nil {
		return
	}

	if blockIfs, ok := p[block_store_name]; ok {
		blocks := blockIfs.(map[string]*BlockDirect)
		blocks[name] = block
		p[block_store_name] = blocks

		return
	}

	blocks := make(map[string]*BlockDirect)
	blocks[name] = block
	p[block_store_name] = blocks
}
