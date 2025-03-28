package uploadlist

import (
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/ucan"
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
		err := b.AssignString(c.Cursor)
		if err != nil {
			return nil, err
		}
		data["cursor"] = b.Build()
	}
	if c.Size != 0 {
		b := basicnode.Prototype.Int.NewBuilder()
		err := b.AssignInt(c.Size)
		if err != nil {
			return nil, err
		}
		data["size"] = b.Build()
	}
	if c.Pre {
		b := basicnode.Prototype.Bool.NewBuilder()
		err := b.AssignBool(c.Pre)
		if err != nil {
			return nil, err
		}
		data["pre"] = b.Build()
	}
	return data, nil
}

func NewCapability(space did.DID, nb *Caveat) ucan.Capability[ucan.MapBuilder] {
	return ucan.NewCapability(Ability, space.String(), ucan.MapBuilder(nb))
}
