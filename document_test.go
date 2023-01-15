package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type Person struct {
	name string
	role *Role
}

func (p *Person) GetName() string {
	return p.name
}

func (p *Person) GetRole() *Role {
	return p.role
}

type Role struct {
	name string
}

func (r *Role) GetName() string {
	return r.name
}

func TestTemplate(t *testing.T) {
	testStringTpl(t)
	testVariableTpl(t)
	testVStructPropertyTpl(t)
	testAdd(t)
	testMulti(t)
	testDiv(t)
	testIf(t)
}

func testStringTpl(t *testing.T) {
	tpl, err := buildTemplate("Hello world")
	assert.Nil(t, err)
	content, err := tpl.execute(nil)
	assert.Nil(t, err)
	assert.Equal(t, "Hello world", content)
}

func testVariableTpl(t *testing.T) {
	tpl, err := buildTemplate("Hello {{ name }}")
	assert.Nil(t, err)
	content, err := tpl.execute(Params{"name": "Jack"})
	assert.Nil(t, err)
	assert.Equal(t, "Hello Jack", content)
}

func testVStructPropertyTpl(t *testing.T) {
	person := &Person{name: "Jack", role: &Role{name: "Admin"}}
	tpl, err := buildTemplate("Hello {{ person.role.name }} {{ person.name }}")
	assert.Nil(t, err)
	content, err := tpl.execute(Params{"person": person})
	assert.Nil(t, err)
	assert.Equal(t, "Hello Admin Jack", content)
}

func testAdd(t *testing.T) {
	tpl, err := buildTemplate("{{ a + b }}")
	assert.Nil(t, err)
	content, err := tpl.execute(Params{"a": 1, "b": 2})
	assert.Nil(t, err)
	assert.Equal(t, "3", content)
	content, err = tpl.execute(Params{"a": 1, "b": -2})
	assert.Nil(t, err)
	assert.Equal(t, "-1", content)
}

func testMulti(t *testing.T) {
	tpl, err := buildTemplate("{{ a * b }}")
	assert.Nil(t, err)
	content, err := tpl.execute(Params{"a": 3, "b": 4})
	assert.Nil(t, err)
	assert.Equal(t, "12", content)
}

func testDiv(t *testing.T) {
	tpl, err := buildTemplate("{{ a / b }}")
	assert.Nil(t, err)
	content, err := tpl.execute(Params{"a": 4, "b": 2})
	assert.Nil(t, err)
	assert.Equal(t, "2", content)
	_, err = tpl.execute(Params{"a": 2, "b": 0})
	assert.ErrorContains(t, err, "can't use 0 as denominator")
}

func testIf(t *testing.T) {
	tpl, err := buildTemplate(`{% if a %}a is true{% else %}a is false{% endif %}`)
	assert.Nil(t, err)
	content, err := tpl.execute(Params{"a": true})
	assert.Nil(t, err)
	assert.Equal(t, "a is true", content)
	content, err = tpl.execute(Params{"a": false})
	assert.Nil(t, err)
	assert.Equal(t, "a is false", content)
	content, err = tpl.execute(Params{"a": 1})
	assert.Nil(t, err)
	assert.Equal(t, "a is true", content)
	content, err = tpl.execute(Params{"a": 0})
	assert.Nil(t, err)
	assert.Equal(t, "a is false", content)
	content, err = tpl.execute(Params{"a": "hello world"})
	assert.Nil(t, err)
	assert.Equal(t, "a is true", content)
	content, err = tpl.execute(Params{"a": ""})
	assert.Nil(t, err)
	assert.Equal(t, "a is false", content)
	content, err = tpl.execute(Params{"a": []int{}})
	assert.Nil(t, err)
	assert.Equal(t, "a is false", content)
	tpl2, err := buildTemplate(`{% if a %}a is true{% elseif b %}b is true{% else %}a and b are false{% endif %}`)
	assert.Nil(t, err)
	content, err = tpl2.execute(Params{"a": true, "b": false})
	assert.Nil(t, err)
	assert.Equal(t, "a is true", content)
	content, err = tpl2.execute(Params{"a": false, "b": true})
	assert.Nil(t, err)
	assert.Equal(t, "b is true", content)
	content, err = tpl2.execute(Params{"a": false, "b": false})
	assert.Nil(t, err)
	assert.Equal(t, "a and b are false", content)
}

// func testFor(t *testing.T) {
// 	var is []int
// 	tpl, err := buildTemplate(`{% for k, v in arr %}{% endfor %}`)
// }
