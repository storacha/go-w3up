package storeadd

import (
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/ucan"
)

const Ability = "store/add"

type Caveat struct {
	Link   ipld.Link
	Size   uint64
	Origin *ipld.Link
}

var _ ucan.MapBuilder = (*Caveat)(nil)

func (c *Caveat) Build() (map[string]datamodel.Node, error) {
	data := map[string]datamodel.Node{}

	b := basicnode.Prototype.Link.NewBuilder()
	err := b.AssignLink(c.Link)
	if err != nil {
		return nil, err
	}
	data["link"] = b.Build()

	b = basicnode.Prototype.Int.NewBuilder()
	err = b.AssignInt(int64(c.Size))
	if err != nil {
		return nil, err
	}
	data["size"] = b.Build()

	if c.Origin != nil {
		b = basicnode.Prototype.Link.NewBuilder()
		err = b.AssignLink(c.Link)
		if err != nil {
			return nil, err
		}
		data["origin"] = b.Build()
	}

	return data, nil
}

func NewCapability(space did.DID, nb *Caveat) ucan.Capability[ucan.MapBuilder] {
	return ucan.NewCapability(Ability, space.String(), ucan.MapBuilder(nb))
}
