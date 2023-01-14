package template

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestParam(t *testing.T) {

}

type Foo struct {
	foo string
	Bar string
}

type Bar struct {
	Foo          *Foo
	Foos         []*Foo
	Bar          map[string]int
	ComplexField map[string][]*Foo
}

func (b *Bar) GetName() string {
	return "Bar"
}

func TestIndex(t *testing.T) {
	var (

	// stringMap = map[string]string{"foo": "Foo"}
	// objMap    = map[string]*Foo{"foo": &Foo{foo: "Foo"}}
	)
	var (
		val reflect.Value
		err error
		foo = &Foo{foo: "Foo"}
	)

	intSlice := []int{1}
	val, err = index(reflect.ValueOf(intSlice), reflect.ValueOf(0))
	assert.Nil(t, err)
	assert.Equal(t, val.Interface(), 1)
	_, err = index(reflect.ValueOf(intSlice), reflect.ValueOf(1))
	assert.EqualError(t, err, "out of boundary, got 1")
	_, err = index(reflect.ValueOf(intSlice), reflect.ValueOf("1"))
	assert.EqualError(t, err, "con't use type string as array or slice index")

	strSlice := []string{"string"}
	val, err = index(reflect.ValueOf(strSlice), reflect.ValueOf(0))
	assert.Nil(t, err)
	assert.Equal(t, val.Interface(), "string")
	_, err = index(reflect.ValueOf(intSlice), reflect.ValueOf(1))
	assert.EqualError(t, err, "out of boundary, got 1")

	objSlice := []*Foo{foo}
	val, err = index(reflect.ValueOf(objSlice), reflect.ValueOf(0))
	assert.Nil(t, err)
	valInterface := val.Interface()
	valObj, ok := valInterface.(*Foo)
	assert.Equal(t, ok, true)
	assert.Equal(t, valObj, foo)
	_, err = index(reflect.ValueOf(intSlice), reflect.ValueOf(1))
	assert.EqualError(t, err, "out of boundary, got 1")

	intMap := map[string]int{"foo": 1, "bar": 2, "test": 3}
	intMapVal := reflect.ValueOf(intMap)
	val, err = index(intMapVal, reflect.ValueOf("foo"))
	assert.Nil(t, err)
	assert.Equal(t, val.Interface(), 1)
	_, err = index(intMapVal, reflect.ValueOf("Bar"))
	assert.ErrorContains(t, err, "index Bar don't exist in map")
	_, err = index(intMapVal, reflect.ValueOf(1))
	assert.EqualError(t, err, "con't use type int as map[string] key")

	stringMap := map[string]string{"foo": "Foo"}
	stringMapVal := reflect.ValueOf(stringMap)
	val, err = index(stringMapVal, reflect.ValueOf("foo"))
	assert.Nil(t, err)
	assert.Equal(t, val.Interface(), "Foo")
	_, err = index(intMapVal, reflect.ValueOf("Bar"))
	assert.ErrorContains(t, err, "index Bar don't exist in map")
	_, err = index(intMapVal, reflect.ValueOf(1))
	assert.EqualError(t, err, "con't use type int as map[string] key")

	objMap := map[string]*Foo{"foo": foo}
	objMapVal := reflect.ValueOf(objMap)
	val, err = index(objMapVal, reflect.ValueOf("foo"))
	assert.Nil(t, err)
	valObj, ok = val.Interface().(*Foo)
	assert.Equal(t, ok, true)
	assert.Equal(t, valObj, foo)
	_, err = index(intMapVal, reflect.ValueOf("Bar"))
	assert.ErrorContains(t, err, "index Bar don't exist in map")
	_, err = index(intMapVal, reflect.ValueOf(1))
	assert.EqualError(t, err, "con't use type int as map[string] key")
}

func TestCall(t *testing.T) {
	var func1 = func(a, b int) int {
		return a + b
	}
	var (
		val reflect.Value
		err error
	)

	val, err = call(reflect.ValueOf(func1), reflect.ValueOf(1), reflect.ValueOf(1))
	assert.Nil(t, err)
	assert.Equal(t, val.Interface(), 2)
	_, err = call(reflect.ValueOf(func1), reflect.ValueOf(1))
	assert.ErrorContains(t, err, "wrong number of args: got 1 want 2")

	var func2 = func(a, b int) (int, error) {
		if b == 0 {
			return 0, errors.Errorf("can't use 0 as denominator")
		}
		return a / b, nil
	}

	val, err = call(reflect.ValueOf(func2), reflect.ValueOf(1), reflect.ValueOf(1))
	assert.Nil(t, err)
	assert.Equal(t, val.Interface(), 1)
	_, err = call(reflect.ValueOf(func2), reflect.ValueOf(1), reflect.ValueOf(0))
	assert.ErrorContains(t, err, "can't use 0 as denominator")
}

func TestProperty(t *testing.T) {
	var foo = &Foo{foo: "Foo", Bar: "bar"}
	_, err := property(reflect.ValueOf(foo), "foo")
	assert.ErrorContains(t, err, "property named foo isn't exported in type template.Foo")
	_, err = property(reflect.ValueOf(foo), "bar")
	assert.ErrorContains(t, err, "property named bar don't exist in type template.Foo")
	val, err := property(reflect.ValueOf(foo), "Bar")
	assert.Nil(t, err)
	assert.Equal(t, val.Interface(), "bar")
}

func TestCalc(t *testing.T) {
	testCalc(t, 1, 2, "+", int64(3))
	testCalc(t, 1, 2, "-", int64(-1))
	testCalc(t, 1, 2, "*", int64(2))
	testCalc(t, 1, 2, "/", float64(0.5))
	_, err := calc(reflect.ValueOf(1), reflect.ValueOf(0), "/")
	assert.ErrorContains(t, err, "can't use 0 as denominator")

	testCalc(t, uint(1), 2, "+", int64(3))
	testCalc(t, uint(1), 2, "-", int64(-1))
	testCalc(t, uint(1), 2, "*", int64(2))
}

func testCalc(t *testing.T, a, b any, op string, expected any) {
	val, err := calc(reflect.ValueOf(a), reflect.ValueOf(b), op)
	assert.Nil(t, err)
	assert.Equal(t, expected, val.Interface())
}

func TestCompare(t *testing.T) {
	var (
		val reflect.Value
		err error
	)
	val, err = eq(reflect.ValueOf(1), reflect.ValueOf(1))
	assert.Nil(t, err)
	assert.Equal(t, true, val.Interface())

	val, err = eq(reflect.ValueOf(1), reflect.ValueOf(2))
	assert.Nil(t, err)
	assert.Equal(t, false, val.Interface())

	val, err = eq(reflect.ValueOf(int32(1)), reflect.ValueOf(1))
	assert.Nil(t, err)
	assert.Equal(t, true, val.Interface())

	val, err = eq(reflect.ValueOf(int32(1)), reflect.ValueOf(1.0))
	assert.Nil(t, err)
	assert.Equal(t, true, val.Interface())

	val, err = eq(reflect.ValueOf(1), reflect.ValueOf("1"))
	assert.Nil(t, err)
	assert.Equal(t, false, val.Interface())

	var (
		a any = 1
		b any = 2
	)
	val, err = eq(reflect.ValueOf(a), reflect.ValueOf(b))
	assert.Nil(t, err)
	assert.Equal(t, false, val.Interface())

	var (
		foo *Foo = &Foo{foo: "foo"}
		bar any  = foo
	)

	val, err = eq(reflect.ValueOf(foo), reflect.ValueOf(bar))
	assert.Nil(t, err)
	assert.Equal(t, true, val.Interface())

	val, err = greater(reflect.ValueOf(1), reflect.ValueOf(1))
	assert.Nil(t, err)
	assert.Equal(t, false, val.Interface())

	val, err = greater(reflect.ValueOf(1), reflect.ValueOf(2))
	assert.Nil(t, err)
	assert.Equal(t, false, val.Interface())

	val, err = greater(reflect.ValueOf(2), reflect.ValueOf(1))
	assert.Nil(t, err)
	assert.Equal(t, true, val.Interface())

	val, err = greater(reflect.ValueOf(int32(2)), reflect.ValueOf(1))
	assert.Nil(t, err)
	assert.Equal(t, true, val.Interface())

	val, err = greater(reflect.ValueOf(int32(2)), reflect.ValueOf(1.0))
	assert.Nil(t, err)
	assert.Equal(t, true, val.Interface())

	val, err = greaterOrEqual(reflect.ValueOf(1), reflect.ValueOf(1))
	assert.Nil(t, err)
	assert.Equal(t, true, val.Interface())

	val, err = greaterOrEqual(reflect.ValueOf(1), reflect.ValueOf(2))
	assert.Nil(t, err)
	assert.Equal(t, false, val.Interface())

	val, err = greaterOrEqual(reflect.ValueOf(2), reflect.ValueOf(1))
	assert.Nil(t, err)
	assert.Equal(t, true, val.Interface())

	val, err = greaterOrEqual(reflect.ValueOf(int32(2)), reflect.ValueOf(1))
	assert.Nil(t, err)
	assert.Equal(t, true, val.Interface())

	val, err = greaterOrEqual(reflect.ValueOf(int32(2)), reflect.ValueOf(1.0))
	assert.Nil(t, err)
	assert.Equal(t, true, val.Interface())

}

func TestGet(t *testing.T) {
	foo := &Foo{foo: "foo", Bar: "bar"}
	foo1 := &Foo{foo: "foo", Bar: "bar"}
	foo2 := &Foo{foo: "foo", Bar: "bar"}
	foo3 := &Foo{foo: "foo", Bar: "bar"}
	bar := &Bar{
		Foo:          foo,
		Foos:         []*Foo{foo},
		Bar:          map[string]int{"1": 1, "2": 2, "3": 3},
		ComplexField: map[string][]*Foo{"key1": {foo1, foo2, foo3}},
	}

	var (
		val reflect.Value
		err error
	)
	val, err = get(bar, "Foo")
	assert.Nil(t, err)
	assert.Equal(t, val.Interface(), foo)

	val, err = get(bar, "Foo", "Bar")
	assert.Nil(t, err)
	assert.Equal(t, val.Interface(), "bar")

	val, err = get(bar, "Bar", "1")
	assert.Nil(t, err)
	assert.Equal(t, val.Interface(), 1)

	_, err = get(bar, "Bar", "4")
	assert.ErrorContains(t, err, "index 4 don't exist in map")

	_, err = get(bar, "Bars")
	assert.NotNil(t, err)

	val, err = get(bar, "ComplexField", "key1", 1)
	assert.Nil(t, err)
	assert.Equal(t, val.Interface(), foo2)

	val, err = get(bar, "Name")
	assert.Nil(t, err)
	assert.Equal(t, val.Interface(), "Bar")
}
