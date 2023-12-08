package uploadlist

import (
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/web3-storage/go-ucanto/did"
	"github.com/web3-storage/go-ucanto/ucan"
)

const Ability = "upload/list"

type Caveat struct {
	Cursor string
	Size   int64
	Pre    bool
}

var _ ucan.MapBuilder = (*Caveat)(nil)

func (c *Caveat) Build() (map[string]datamodel.Node, error) {
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

func NewCapability(space did.DID, nb *Caveat) ucan.Capability[ucan.MapBuilder] {
	return ucan.NewCapability(Ability, space.String(), ucan.MapBuilder(nb))
}
