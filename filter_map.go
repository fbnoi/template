package template

import (
	"reflect"
	"sync"

	"github.com/pkg/errors"
)

var (
	filter_map = &filterMap{
		store:  buildInFuncs(),
		locker: &sync.RWMutex{},
	}
)

type filterMap struct {
	store  map[string]reflect.Value
	locker *sync.RWMutex
}

func RegisterFilter(name string, fn any) error {
	if fn == nil {
		return nil
	}
	if !goodName(name) {
		return errors.Errorf("can't use %s as filter's name", name)
	}
	fnValue := reflect.ValueOf(fn)
	if fnValue.Kind() != reflect.Func {
		return errors.Errorf("can't register %s as filter", fnValue.Kind())
	}
	if !goodFilter(fnValue.Type()) {
		return errors.Errorf("filter accept 1 arg and return 1 or 2 value(s)")
	}
	filter_map.locker.Lock()
	defer filter_map.locker.Unlock()
	filter_map.store[name] = fnValue

	return nil
}

func goodFilter(typ reflect.Type) bool {
	if typ == nil {
		return false
	}
	if typ.Kind() != reflect.Func {
		return false
	}
	if !goodFunc(typ) {
		return false
	}
	if typ.NumIn() != 1 {
		return false
	}

	return true
}
