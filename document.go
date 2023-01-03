package template

import (
	"strings"
	"sync"

	"github.com/pkg/errors"
)

var (
	_store = &Documents{
		store:  make(map[string]*Document),
		locker: &sync.RWMutex{},
	}
)

func NewDocument() *Document {
	return &Document{blocks: make(map[string]*BlockDirect)}
}

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
	blocks map[string]*BlockDirect

	extended bool
}

func (doc *Document) Block(name string) *BlockDirect {
	if block, ok := doc.blocks[name]; ok {
		return block
	}

	return nil
}

// TODO: walk
func (doc *Document) Execute(p Params) (string, error) {
	sb := &strings.Builder{}
	nd := doc
	if doc.Extend != nil {
		nd = doc.Extend.Doc
		for n, b := range doc.blocks {
			p.setBlock(n, b)
		}
	}
	for _, v := range nd.Body.List {
		if str, err := v.Execute(p); err != nil {
			return "", err
		} else {
			sb.WriteString(str)
		}
	}

	return sb.String(), nil
}

func (doc *Document) Append(x Direct) {
	if doc.Body == nil {
		doc.Body = &SectionDirect{}
	}
	doc.Body.List = append(doc.Body.List, x)
}

func (d *Document) Validate() error {
	if d.Extend != nil {
		if err := d.Extend.Validate(); err != nil {
			return err
		}
	}

	return d.Body.Validate()
}
