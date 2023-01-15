package template

import (
	"reflect"
	"sync"

	"github.com/pkg/errors"
)

var (
	func_map = make(map[string]reflect.Value)
	locker   = sync.Mutex{}
)

func RegisterFunc(name string, fn any) error {
	if fn == nil {
		return nil
	}
	if !goodName(name) {
		return errors.Errorf("can't use %s as func's name", name)
	}
	if _, ok := func_map[name]; ok {
		return errors.Errorf("func named %s already exists", name)
	}
	fnValue := reflect.ValueOf(fn)
	if fnValue.Kind() != reflect.Func {
		return errors.Errorf("can't register %s as func", fnValue.Kind())
	}
	if !goodFunc(fnValue.Type()) {
		return errors.Errorf("func return %d values; should be 1 or 2", fnValue.Type().NumOut())
	}
	locker.Lock()
	defer locker.Unlock()
	func_map[name] = fnValue

	return nil
}
