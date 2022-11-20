package template

import (
	"reflect"
	"testing"
)

type Foo struct {
	bar  string
	Bar1 string
}

func (f *Foo) Bar() string {
	return f.bar
}

func TestParam(t *testing.T) {
	p := &Params{store: make(map[string]any)}
	p.Set("count", 10)
	p.Set("data", &Foo{
		bar:  "bar",
		Bar1: "Bar1",
	})

	if val, err := Get("count", p); err != nil {
		t.Error(err)
	} else if v, ok := val.(int); !ok {
		t.Errorf("expect type int, got %s", reflect.TypeOf(val).Name())
	} else {
		t.Log(v)
	}

	if val, err := GetDot("data.bar", p); err != nil {
		t.Error(err)
	} else if v, ok := val.(string); !ok {
		t.Errorf("expect type string, got %s", reflect.TypeOf(val).Name())
	} else {
		t.Log(v)
	}

	if val, err := GetDot("data.Bar1", p); err != nil {
		t.Error(err)
	} else if v, ok := val.(string); !ok {
		t.Errorf("expect type string, got %s", reflect.TypeOf(val).Name())
	} else if v != "Bar1" {
		t.Logf("expect Bar1, got %s", v)
	}

}
