package template

import (
	"reflect"
	"sync"

	"github.com/pkg/errors"
)

var (
	func_map = &funcMap{
		store:  make(map[string]reflect.Value),
		locker: &sync.RWMutex{},
	}
)

type funcMap struct {
	store  map[string]reflect.Value
	locker *sync.RWMutex
}

func RegisterFunc(name string, fn any) error {
	if fn == nil {
		return nil
	}
	if !goodName(name) {
		return errors.Errorf("can't use %s as func's name", name)
	}
	fnValue := reflect.ValueOf(fn)
	if fnValue.Kind() != reflect.Func {
		return errors.Errorf("can't register %s as func", fnValue.Kind())
	}
	if !goodFunc(fnValue.Type()) {
		return errors.Errorf("func return %d values; should be 1 or 2", fnValue.Type().NumOut())
	}
	func_map.locker.Lock()
	defer func_map.locker.Unlock()
	func_map.store[name] = fnValue

	return nil
}

func getFunc(name string) reflect.Value {
	func_map.locker.RLock()
	defer func_map.locker.RUnlock()

	if fn, ok := func_map.store[name]; ok {
		return fn
	}

	return zeroValue
}
