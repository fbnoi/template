package template

import (
	"log"
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
	doc, err := BuildTemplate("Hello world")
	assert.Nil(t, err)
	content, err := doc.Execute(nil)
	assert.Nil(t, err)
	assert.Equal(t, "Hello world", content)

	doc, err = BuildTemplate("Hello {{ name }}")
	assert.Nil(t, err)
	content, err = doc.Execute(Params{"name": "Jack"})
	assert.Nil(t, err)
	assert.Equal(t, "Hello Jack", content)

	person := &Person{name: "Jack", role: &Role{name: "Admin"}}
	doc, err = BuildTemplate("Hello {{ person.role.name }} {{ person.name }}")
	assert.Nil(t, err)
	content, err = doc.Execute(Params{"person": person})
	assert.Nil(t, err)
	assert.Equal(t, "Hello Admin Jack", content)

	doc, err = BuildTemplate("Hello {{ person['role']['name'] }} {{ person['name'] }}")
	assert.Nil(t, err)
	content, err = doc.Execute(Params{"person": person})
	assert.Nil(t, err)
	assert.Equal(t, "Hello Admin Jack", content)
}

func TestFileTemplate(t *testing.T) {
	doc, err := BuildFileTemplate("../var/template/block_test.html.tpl")
	assert.Nil(t, err)
	log.Print(doc)
}
