package client

import (
	"context"

	cmdspace "github.com/fil-forge/libforge/commands/space"
	"github.com/fil-forge/ucantone/did"
)

// SpaceInfo invokes the space/info capability to get information about a space,
// including which providers are associated with it.
func (c *Client) SpaceInfo(ctx context.Context, space did.DID) (cmdspace.InfoOK, error) {
	var ok cmdspace.InfoOK

	err := invokeExecuteAndUnpack(
		ctx,
		c,
		space,
		cmdspace.Info,
		&cmdspace.InfoArguments{},
		&ok,
	)
	if err != nil {
		return cmdspace.InfoOK{}, err
	}
	return ok, err
}
