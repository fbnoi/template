package template

import (
	"reflect"
	"sync"

	"github.com/pkg/errors"
)

var (
	filter_map = &filterMap{
		store:  buildInFilters(),
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
	if !goodFunc(fnValue.Type()) {
		return errors.Errorf("filter return %d values; should be 1 or 2", fnValue.Type().NumOut())
	}
	filter_map.locker.Lock()
	defer filter_map.locker.Unlock()

	filter_map.store[name] = fnValue

	return nil
}

func getFilter(name string) reflect.Value {
	filter_map.locker.RLock()
	defer filter_map.locker.RUnlock()

	if fn, ok := filter_map.store[name]; ok {
		return fn
	}

	return zeroValue
}
