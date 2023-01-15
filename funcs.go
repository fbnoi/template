package template

import "reflect"

var funcs = map[string]reflect.Value{
	"PS": reflect.ValueOf(PS),
	"P":  reflect.ValueOf(P),
}

func buildInFuncs() map[string]reflect.Value {
	return funcs
}

func PS(ps ...Params) Params {
	p := Params{}
	for _, p := range ps {
		for k, v := range p {
			p[k] = v
		}
	}

	return p
}

func P(k string, v any) Params {
	return Params{k: v}
}
