package template

import (
	"sync"

	"github.com/pkg/errors"
)

var (
	_store = &Documents{
		store:  make(map[string]*Document),
		locker: &sync.RWMutex{},
	}
)

type Documents struct {
	store  map[string]*Document
	locker *sync.RWMutex
}

func (docs *Documents) AddDoc(name string, doc *Document) error {
	docs.locker.Lock()
	defer docs.locker.Unlock()
	if _, ok := docs.store[name]; ok {
		return errors.Errorf("document with name [%s] has already exists.", name)
	}

	docs.store[name] = doc

	return nil
}

func (docs *Documents) Doc(name string) *Document {
	docs.locker.RLock()
	defer docs.locker.RUnlock()

	if doc, ok := docs.store[name]; ok {
		return doc
	}

	return nil
}

type Document struct {
	Extend *ExtendDirect
	Body   *SectionDirect
}

func (doc *Document) Block(name string) *BlockDirect {
	for _, br := range doc.Body.List {
		if b, ok := br.(*BlockDirect); ok && b.Name.Value.Value() == name {
			return b
		}
	}

	return nil
}

func (doc *Document) Execute(data any) (string, error) {
	if doc.Extend != nil {
		if pDoc := _store.Doc(doc.Extend.Path.Value.Value()); pDoc != nil {
			return pDoc.executeWithTpl(data, doc)
		}
	}

	return doc.execute(data)
}

func (doc *Document) execute(data any) (string, error) {
	return "", nil
}

func (doc *Document) executeWithTpl(data any, xDoc *Document) (string, error) {
	return "", nil
}

func (doc *Document) Append(x Direct) {
	if doc.Body == nil {
		doc.Body = &SectionDirect{}
	}
	doc.Body.List = append(doc.Body.List, x)
}

func (*Document) directNode() {}

func (d *Document) Validate() error {
	if d.Extend != nil {
		if err := d.Extend.Validate(); err != nil {
			return err
		}
	}

	return d.Body.Validate()
}
