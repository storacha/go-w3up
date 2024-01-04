package storeadd

import (
	_ "embed"

	"github.com/ipld/go-ipld-prime"
)

//go:embed result.ipldsch
var ResultSchema []byte

type Success struct {
	// Status of the item to store. A "done" status indicates that it is not
	// necessary to upload the item. An "upload" status indicates that the item
	// should be uploaded to the provided URL.
	Status string
	// With is the DID of the space this item will be stored in.
	With string
	// Link is the CID of the item.
	Link    ipld.Link
	Url     *string
	Headers *Headers
	// Allocated is the total bytes allocated in the space to accommodate this
	// stored item. May be zero if the item is _already_ stored in _this_ space.
	Allocated uint64
}

type Headers struct {
	Keys   []string
	Values map[string]string
}

type Failure struct {
	Name    *string
	Message string
	Stack   *string
}
