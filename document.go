package template

import (
	"strings"
	"sync"

	"github.com/pkg/errors"
)

var (
	_cache = &documents{
		cache:  make(map[string]*Document),
		locker: &sync.RWMutex{},
	}
)

func newDocument() *Document {
	return &Document{blocks: make(map[string]*blockDirect)}
}

type documents struct {
	cache  map[string]*Document
	locker *sync.RWMutex
}

func (docs *documents) addDoc(name string, doc *Document) error {
	docs.locker.Lock()
	defer docs.locker.Unlock()
	if _, ok := docs.cache[name]; ok {
		return errors.Errorf("document with name [%s] has already exists.", name)
	}

	docs.cache[name] = doc

	return nil
}

func (docs *documents) doc(name string) *Document {
	docs.locker.RLock()
	defer docs.locker.RUnlock()

	if doc, ok := docs.cache[name]; ok {
		return doc
	}

	return nil
}

type Document struct {
	extend *extendDirect
	body   *sectionDirect
	blocks map[string]*blockDirect

	extended bool
}

func (doc *Document) Block(name string) *blockDirect {
	if block, ok := doc.blocks[name]; ok {
		return block
	}

	return nil
}

func (doc *Document) execute(p Params) (string, error) {
	sb := &strings.Builder{}
	nd := doc
	if doc.extend != nil {
		nd = doc.extend.doc
		for n, b := range doc.blocks {
			p.setBlock(n, b)
		}
	}
	for _, v := range nd.body.list {
		if str, err := v.execute(p); err != nil {
			return "", err
		} else {
			sb.WriteString(str)
		}
	}

	return sb.String(), nil
}

func (doc *Document) append(x direct) {
	if doc.body == nil {
		doc.body = &sectionDirect{}
	}
	doc.body.list = append(doc.body.list, x)
}

func (d *Document) validate() error {
	if d.extend != nil {
		if err := d.extend.validate(); err != nil {
			return err
		}
	}

	return d.body.validate()
}
