package capability

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/ipld/go-ipld-prime/schema"
)

type UploadListCaveat struct {
	Cursor string
	Size   int64
	Pre    bool
}

func (c *UploadListCaveat) Build() (map[string]datamodel.Node, error) {
	data := map[string]datamodel.Node{}
	if c.Cursor != "" {
		b := basicnode.Prototype.String.NewBuilder()
		b.AssignString(c.Cursor)
		data["cursor"] = b.Build()
	}
	if c.Size != 0 {
		b := basicnode.Prototype.Int.NewBuilder()
		b.AssignInt(c.Size)
		data["size"] = b.Build()
	}
	if c.Pre {
		b := basicnode.Prototype.Bool.NewBuilder()
		b.AssignBool(c.Pre)
		data["pre"] = b.Build()
	}
	return data, nil
}

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
	Name    string
	Message string
	Stack   string
}
