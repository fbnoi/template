package test

import "testing"

func TestParam(t *testing.T) {

}

type Foo struct {
	foo string
}

func TestIndex(t *testing.T) {
	var (
		intSlice  = []int{1, 2, 3, 4, 5}
		strSlice  = []string{"1", "2", "3", "4", "5"}
		objSlice  = []*Foo{&Foo{foo: "Foo"}, &Foo{foo: "Bar"}}
		intMap    = map[string]int{"foo": 1, "bar": 2}
		stringMap = map[string]string{"foo": "Foo", "bar": "Bar"}
	)
}
