package storeadd

import (
	_ "embed"

	"github.com/ipld/go-ipld-prime"
)

//go:embed result.ipldsch
var ResultSchema []byte

type Success struct {
	Status  string
	With    string
	Link    ipld.Link
	Url     *string
	Headers *Headers
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
