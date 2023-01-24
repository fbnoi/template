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
	// testStringTpl(t)
	// testVariableTpl(t)
	// testVStructPropertyTpl(t)
	// testAdd(t)
	// testMulti(t)
	// testDiv(t)
	// testIf(t)
	// testFor(t)
	// testSet(t)
	// testFunc(t)
	// testCache(t)
	testFileTpl(t)
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

func testFor(t *testing.T) {
	tpl, err := buildTemplate(`{% for k, v in arr %}{{ k }}{{ v }}{% endfor %}`)
	assert.Nil(t, err)
	content, err := tpl.execute(Params{"arr": []int{1, 2, 4}})
	assert.Nil(t, err)
	assert.Equal(t, "011224", content)

	tpl, err = buildTemplate(`{% for _, v in arr %}{{ v }}{% endfor %}`)
	assert.Nil(t, err)
	content, err = tpl.execute(Params{"arr": []int{1, 2, 4}})
	assert.Nil(t, err)
	assert.Equal(t, "124", content)

	tpl, err = buildTemplate(`{% for v in arr %}{{ v }}{% endfor %}`)
	assert.Nil(t, err)
	content, err = tpl.execute(Params{"arr": []int{1, 2, 4}})
	assert.Nil(t, err)
	assert.Equal(t, "124", content)
}

func testSet(t *testing.T) {
	tpl, err := buildTemplate(`{% set a = true %}{{ a }}`)
	assert.Nil(t, err)
	content, err := tpl.execute(nil)
	assert.Nil(t, err)
	assert.Equal(t, "true", content)
}

func testFunc(t *testing.T) {
	greeting := func(name string) string {
		return "Hello " + name
	}
	err := RegisterFunc("greeting", greeting)
	assert.Nil(t, err)
	tpl, err := buildTemplate(`{{ greeting(name) }}`)
	assert.Nil(t, err)
	content, err := tpl.execute(Params{"name": "John"})
	assert.Nil(t, err)
	assert.Equal(t, "Hello John", content)
}

func testCache(t *testing.T) {
	_, err := buildTemplate(`cache`)
	assert.Nil(t, err)
	_, ok := _cache.cache[abstract([]byte("cache"))]
	assert.Equal(t, true, ok)
}

func testFileTpl(t *testing.T) {
	tpl, err := buildFileTemplate("./var/block_test.html.tpl")
	assert.Nil(t, err)
	assert.NotNil(t, _cache.doc("./var/block_test.html.tpl"))
	assert.NotNil(t, _cache.doc("./var/base.html.tpl"))
	assert.NotNil(t, _cache.doc("./var/include_test.html.tpl"))
	content, err := tpl.execute(Params{
		"some_content":  "content in base tpl",
		"show_content1": true, "content1": "show content1",
		"show_content2": false, "content2": "show content2",
		"show_content3": true, "content3": "show content3",
		"show_content4": true, "content4": "show content4",
		"list":         map[string]string{"key1": "value1", "key2": "value2"},
		"content_with": "Hello include",
	})
	assert.Nil(t, err)
	assert.Contains(t, content, "content in base tpl")
	assert.Contains(t, content, "show content1")
	assert.Contains(t, content, "show content3")
	assert.Contains(t, content, "not show content2")
	assert.Contains(t, content, "show content3 and show content4")
	assert.NotContains(t, content, "show content3 and not show content4")
	assert.Contains(t, content, "key1:value1")
	assert.Contains(t, content, "key2:value2")
	assert.Contains(t, content, "some text in block page")
	assert.Contains(t, content, "some content in include tpl")
	assert.Contains(t, content, "show content4")
	assert.NotContains(t, content, "Hello include")
}
