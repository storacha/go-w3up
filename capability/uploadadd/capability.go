package uploadadd

import (
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/ucan"
)

const Ability = "upload/add"

type Caveat struct {
	Root   ipld.Link
	Shards []ipld.Link
}

var _ ucan.MapBuilder = (*Caveat)(nil)

func (c *Caveat) Build() (map[string]datamodel.Node, error) {
	data := map[string]datamodel.Node{}

	b := basicnode.Prototype.Link.NewBuilder()
	err := b.AssignLink(c.Root)
	if err != nil {
		return nil, err
	}
	data["root"] = b.Build()

	if c.Shards != nil {
		b := basicnode.Prototype.Any.NewBuilder()
		la, err := b.BeginList(int64(len(c.Shards)))
		if err != nil {
			return nil, err
		}
		for _, s := range c.Shards {
			err := la.AssembleValue().AssignLink(s)
			if err != nil {
				return nil, err
			}
		}
		la.Finish()
		data["shards"] = b.Build()
	}
	return data, nil
}

func NewCapability(space did.DID, nb *Caveat) ucan.Capability[ucan.MapBuilder] {
	return ucan.NewCapability(Ability, space.String(), ucan.MapBuilder(nb))
}
