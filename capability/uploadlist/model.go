package uploadlist

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/schema"
)

//go:embed upload.ipldsch
var UploadSchema []byte

var (
	once sync.Once
	ts   *schema.TypeSystem
	err  error
)

func mustLoadSchema() *schema.TypeSystem {
	once.Do(func() {
		ts, err = ipld.LoadSchemaBytes(UploadSchema)
	})
	if err != nil {
		panic(fmt.Errorf("failed to load IPLD schema: %s", err))
	}
	return ts
}

func UploadListResultType() schema.Type {
	return mustLoadSchema().TypeByName("Result")
}

type UploadListSuccess struct {
	Results []UploadListItem
	Before  *string
	After   *string
	Cursor  *string
	Size    uint64
}

type UploadListItem struct {
	Root       ipld.Link
	Shards     []ipld.Link
	InsertedAt string
	UpdatedAt  string
}

type UploadListFailure struct {
	Name    *string
	Message string
	Stack   *string
}
