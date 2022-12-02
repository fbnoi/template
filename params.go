package template

type Params map[string]any

func (p Params) copy() Params {
	np := make(Params)
	for k, v := range p {
		np[k] = v
	}

	return np
}
