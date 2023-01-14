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
	doc, err := buildTemplate("Hello world")
	assert.Nil(t, err)
	content, err := doc.execute(nil)
	assert.Nil(t, err)
	assert.Equal(t, "Hello world", content)

	doc, err = buildTemplate("Hello {{ name }}")
	assert.Nil(t, err)
	content, err = doc.execute(Params{"name": "Jack"})
	assert.Nil(t, err)
	assert.Equal(t, "Hello Jack", content)

	person := &Person{name: "Jack", role: &Role{name: "Admin"}}
	doc, err = buildTemplate("Hello {{ person.role.name }} {{ person.name }}")
	assert.Nil(t, err)
	content, err = doc.execute(Params{"person": person})
	assert.Nil(t, err)
	assert.Equal(t, "Hello Admin Jack", content)

	doc, err = buildTemplate("Hello {{ person['role']['name'] }} {{ person['name'] }}")
	assert.Nil(t, err)
	content, err = doc.execute(Params{"person": person})
	assert.Nil(t, err)
	assert.Equal(t, "Hello Admin Jack", content)
}

func TestFileTemplate(t *testing.T) {
	doc, err := buildFileTemplate("../var/template/block_test.html.tpl")
	assert.Nil(t, err)
	log.Print(doc)
}
