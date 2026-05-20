package client

import (
	"fmt"

	"github.com/fil-forge/ucantone/principal"
	"github.com/fil-forge/ucantone/ucan"
	"github.com/storacha/guppy/pkg/agentstore"
)

// Option is an option configuring a Client.
type Option func(c *Client) error

// WithUcantoneConnection configures the underlying Ucantone client for the
// Guppy client to use. If one is not provided, [DefaultUcantoneClient] will be
// used.
func WithUcantoneConnection(ucantoneClient UcantoneClient) Option {
	return func(c *Client) error {
		c.ucantoneClient = ucantoneClient
		return nil
	}
}

// TK: Retrieval
// WithReceiptsClient configures the client to use for fetching receipts.
// func WithReceiptsClient(receiptsClient *receipt.Client) Option {
// 	return func(c *Client) error {
// 		c.receiptsClient = receiptsClient
// 		return nil
// 	}
// }

// WithStore configures the agent store for the client to use. If one is not
// provided, a new memory store will be created.
func WithStore(store agentstore.Store) Option {
	return func(c *Client) error {
		c.store = store
		return nil
	}
}

// WithPrincipal configures the principal for the client to use. If a store
// is already configured, the principal will be set on that store. Otherwise,
// a new memory store will be created with this principal.
func WithPrincipal(p principal.Signer) Option {
	return func(c *Client) error {
		// If no store exists yet, create a memory store
		if c.store == nil {
			store, err := agentstore.NewMemory()
			if err != nil {
				return fmt.Errorf("creating memory store: %w", err)
			}
			c.store = store
		}
		// Set the principal on the store
		return c.store.SetPrincipal(p)
	}
}

// WithAdditionalProofs adds proofs to the client that will be included in
// Proofs() results but will not be saved to the client's store. This is
// useful for proofs that are only needed for a single operation.
func WithAdditionalProofs(proofs ...ucan.Delegation) Option {
	return func(c *Client) error {
		c.additionalProofs = append(c.additionalProofs, proofs...)
		return nil
	}
}

// TK: Retrieval
// func WithRetrievalOptions(retrievalOpts ...rclient.Option) Option {
// 	return func(c *Client) error {
// 		c.retrievalOpts = append(c.retrievalOpts, retrievalOpts...)
// 		return nil
// 	}
// }
