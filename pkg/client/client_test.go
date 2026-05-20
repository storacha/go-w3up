package client_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fil-forge/libforge/commands/blob"
	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/testutil"
	"github.com/fil-forge/ucantone/ucan"
	"github.com/fil-forge/ucantone/ucan/delegation"

	"github.com/storacha/guppy/pkg/agentstore"
	"github.com/storacha/guppy/pkg/client"
)

func TestReset(t *testing.T) {
	store := testutil.Must(agentstore.NewMemory())(t)
	c := testutil.Must(client.NewClient(client.WithStore(store)))(t)
	res, err := c.Proofs()
	require.NoError(t, err)
	require.Empty(t, res, "expected no proofs to be present initially")

	issuer := c.Issuer()

	// Some arbitrary delegation
	del := testutil.Must(blob.Add.Delegate(
		c.Issuer(),
		c.Issuer().DID(),
		c.Issuer().DID(),
	))(t)

	err = c.AddProofs(del)
	require.NoError(t, err)
	res, err = c.Proofs()
	require.NoError(t, err)
	require.Equal(t, []ucan.Delegation{del}, res, "expected one proof to be added")

	err = c.Reset()
	require.NoError(t, err, "expected reset to succeed")
	res, err = c.Proofs()
	require.NoError(t, err)
	require.Empty(t, res, "expected all proofs to be removed after reset")
	require.Equal(t, c.DID(), issuer.DID(), "expected issuer to remain unchanged after reset")

	// Verify store state
	storedPrincipal, err := store.Principal()
	require.NoError(t, err)
	require.Equal(t, storedPrincipal.DID(), issuer.DID(), "expected saved principal to be the issuer")

	storedDelegations, err := store.Delegations()
	require.NoError(t, err)
	require.Empty(t, storedDelegations, "expected saved proofs to be empty")
}

func TestProofs(t *testing.T) {
	c := testutil.Must(client.NewClient())(t)

	// Create delegations of different capabilities
	crankDel := testutil.Must(delegation.Delegate(
		c.Issuer(),
		c.Issuer().DID(),
		c.Issuer().DID(),
		"/widget/crank",
	))(t)

	resetDel := testutil.Must(delegation.Delegate(
		c.Issuer(),
		c.Issuer().DID(),
		c.Issuer().DID(),
		"/widget/reset",
	))(t)

	// Create an expired delegation
	expiredDel := testutil.Must(delegation.Delegate(
		c.Issuer(),
		c.Issuer().DID(),
		c.Issuer().DID(),
		"/widget/crank",
		delegation.WithExpiration(ucan.Now()-100), // Expired 100 seconds ago
	))(t)

	// Create a delegation that's not yet valid
	futureDel := testutil.Must(delegation.Delegate(
		c.Issuer(),
		c.Issuer().DID(),
		c.Issuer().DID(),
		"/widget/crank",
		delegation.WithNotBefore(ucan.Now()+100), // Valid 100 seconds from now
	))(t)

	err := c.AddProofs(crankDel, resetDel, expiredDel, futureDel)
	require.NoError(t, err)

	t.Run("no query returns all non-expired, valid delegations", func(t *testing.T) {
		proofs, err := c.Proofs()
		require.NoError(t, err)
		require.ElementsMatch(t, []ucan.Delegation{crankDel, resetDel}, proofs, "should return 2 non-expired delegations")
	})

	t.Run("query by specific command", func(t *testing.T) {
		proofs, err := c.Proofs(agentstore.CapabilityQuery{
			Cmd: "/widget/crank",
		})
		require.NoError(t, err)
		require.ElementsMatch(t, []ucan.Delegation{crankDel}, proofs, "should return 1 /widget/crank delegation")
	})

	t.Run("query by specific subject", func(t *testing.T) {
		proofs, err := c.Proofs(agentstore.CapabilityQuery{
			Sub: c.Issuer().DID(),
		})
		require.NoError(t, err)
		require.ElementsMatch(t, []ucan.Delegation{crankDel}, proofs, "should return 1 delegation")
	})

	t.Run("multiple queries", func(t *testing.T) {
		proofs, err := c.Proofs(
			agentstore.CapabilityQuery{Cmd: "/widget/crank", Sub: c.Issuer().DID()},
			agentstore.CapabilityQuery{Cmd: "/widget/reset", Sub: c.Issuer().DID()},
		)
		require.NoError(t, err)
		require.ElementsMatch(t, []ucan.Delegation{crankDel, resetDel}, proofs, "should return delegations matching either query")
	})

	t.Run("non-matching query", func(t *testing.T) {
		proofs, err := c.Proofs(agentstore.CapabilityQuery{
			Cmd: "/nonexistent/command",
			Sub: c.Issuer().DID(),
		})
		require.NoError(t, err)
		require.Empty(t, proofs, "should return no delegations for non-matching query")
	})

	t.Run("excludes expired delegations", func(t *testing.T) {
		// Expired delegations should be excluded
		proofs, err := c.Proofs()
		require.NoError(t, err)
		require.NotContains(t, proofs, expiredDel, "should not include expired delegation")
	})

	t.Run("excludes future delegations", func(t *testing.T) {
		// Future delegations should be excluded
		proofs, err := c.Proofs()
		require.NoError(t, err)
		require.NotContains(t, proofs, futureDel, "should not include future delegation")
	})

	// TK: Attestations
	// t.Run("session proofs", func(t *testing.T) {
	// 	c := testutil.Must(client.NewClient())(t)

	// 	// Create another principal that will issue the original authorization
	// 	issuer := testutil.RandomSigner()

	// 	// Create an authorization delegation from issuer to client
	// 	authDel := testutil.Must(blob.Add.Delegate(
	// 		issuer,
	// 		c.Issuer().DID(),
	// 		c.Issuer().DID(),
	// 	))(t)

	// 	// Create a session proof (ucan/attest) that attests to the authorization
	// 	sessionProof := testutil.Must(ucancap.Attest.Delegate(
	// 		issuer,
	// 		c.Issuer(),
	// 		issuer.DID(),
	// 	))(t)

	// 	// Add both to the client
	// 	err := c.AddProofs(authDel, sessionProof)
	// 	require.NoError(t, err)

	// 	t.Run("includes session proofs with authorization", func(t *testing.T) {
	// 		// Query for proofs - should get both the authorization and its session proof
	// 		proofs, err := c.Proofs()
	// 		require.NoError(t, err)
	// 		require.ElementsMatch(t, []ucan.Delegation{authDel, sessionProof}, proofs,
	// 			"should return both authorization and session proof")
	// 	})

	// 	t.Run("includes session proofs when querying by capability", func(t *testing.T) {
	// 		// Query by specific capability - should get both the matching authorization and its session proof
	// 		proofs, err := c.Proofs(agentstore.CapabilityQuery{
	// 			Cmd: "upload/add",
	// 			Sub: c.Issuer().DID(),
	// 		})
	// 		require.NoError(t, err)
	// 		require.ElementsMatch(t, []ucan.Delegation{authDel, sessionProof}, proofs,
	// 			"should return authorization and its session proof when querying by capability")
	// 	})

	// 	t.Run("excludes expired session proofs", func(t *testing.T) {
	// 		c := testutil.Must(client.NewClient())(t)

	// 		// Create another principal that will issue the original authorization
	// 		issuer := testutil.Must(signer.Generate())(t)

	// 		// Create an authorization delegation
	// 		authDel := testutil.Must(blob.Add.Delegate(
	// 			issuer,
	// 			c.Issuer(),
	// 			c.Issuer().DID(),
	// 			uploadcap.AddCaveats{Root: testutil.RandomCID(t), Shards: nil},
	// 		))(t)

	// 		// Create an expired session proof
	// 		expiredSessionProof := testutil.Must(ucancap.Attest.Delegate(
	// 			issuer,
	// 			c.Issuer(),
	// 			issuer.DID().String(),
	// 			ucancap.AttestCaveats{Proof: authDel.Link()},
	// 			delegation.WithExpiration(ucan.Now()-100), // Expired
	// 		))(t)

	// 		err := c.AddProofs(authDel, expiredSessionProof)
	// 		require.NoError(t, err)

	// 		// Should only return the authorization, not the expired session proof
	// 		proofs, err := c.Proofs()
	// 		require.NoError(t, err)
	// 		require.ElementsMatch(t, []ucan.Delegation{authDel}, proofs,
	// 			"should exclude expired session proofs")
	// 	})
	// })

	t.Run("command prefix matching", func(t *testing.T) {
		c := testutil.Must(client.NewClient())(t)

		// Create delegations with specific and wildcard capabilities (all with uCmd: * resource)
		specificDel, err := delegation.Delegate(
			c.Issuer(),
			c.Issuer().DID(),
			did.Undef,
			"/widget/crank",
		)
		require.NoError(t, err)

		// Create a delegation with a namespace wildcard capability (upload/*)
		prefixDel, err := delegation.Delegate(
			c.Issuer(),
			c.Issuer().DID(),
			did.Undef,
			"/widget",
		)
		require.NoError(t, err)

		// Create a delegation with a global wildcard capability (*)
		topDel, err := delegation.Delegate(
			c.Issuer(),
			c.Issuer().DID(),
			did.Undef,
			"/",
		)
		require.NoError(t, err)

		err = c.AddProofs(specificDel, prefixDel, topDel)
		require.NoError(t, err)

		t.Run("specific query matches exact, namespace prefix, and top", func(t *testing.T) {
			// Searching for /widget/crank should find:
			// - /widget/crank (exact match)
			// - /widget (prefix match)
			// - / (prefix match)
			proofs, err := c.Proofs(agentstore.CapabilityQuery{
				Cmd: "/widget/crank",
			})
			require.NoError(t, err)
			require.ElementsMatch(t, []ucan.Delegation{specificDel, prefixDel, topDel}, proofs,
				"should find exact match, namespace prefix, and top")
		})

		t.Run("namespace prefix query only matches namespace and top", func(t *testing.T) {
			// Searching for /widget should find:
			// - /widget (exact match)
			// - / (top)
			// NOT /widget/crank (too specific)
			proofs, err := c.Proofs(agentstore.CapabilityQuery{
				Cmd: "/widget",
			})
			require.NoError(t, err)
			require.ElementsMatch(t, []ucan.Delegation{prefixDel, topDel}, proofs,
				"should find namespace and top, not specific commands")
		})

		t.Run("top query only matches top capability", func(t *testing.T) {
			// Searching for / should only match delegations with / capability
			proofs, err := c.Proofs(agentstore.CapabilityQuery{
				Cmd: "/",
			})
			require.NoError(t, err)
			require.ElementsMatch(t, []ucan.Delegation{topDel}, proofs,
				"should only match global prefix capability when query is /")
		})
	})

	t.Run("powerline matching", func(t *testing.T) {
		c := testutil.Must(client.NewClient())(t)
		space := testutil.RandomDID(t)

		// Create a delegation with a specific subject (space DID)
		subjectDel := testutil.Must(delegation.Delegate(
			c.Issuer(),
			c.Issuer().DID(),
			space,
			"/widget/crank",
		))(t)

		// Create a delegation with a powerline
		powerlineDel := testutil.Must(delegation.Delegate(
			c.Issuer(),
			c.Issuer().DID(),
			did.Undef,
			"/widget/crank",
		))(t)

		err = c.AddProofs(subjectDel, powerlineDel)
		require.NoError(t, err)

		t.Run("specific subject query matches exact and powerlines", func(t *testing.T) {
			// Searching for a specific subject should find:
			// - delegations with that exact subject
			// - delegations with uCmd: * (matches any subject)
			proofs, err := c.Proofs(agentstore.CapabilityQuery{
				Cmd: "/widget/crank",
				Sub: space,
			})
			require.NoError(t, err)
			require.ElementsMatch(t, []ucan.Delegation{subjectDel, powerlineDel}, proofs,
				"should match both exact subject and wildcard subject")
		})

		t.Run("empty subject query only matches powerlines", func(t *testing.T) {
			// Searching for empty subject should only match delegations with empty
			// subject (powerlines) NOT delegations with specific subjects (they're
			// too specific)
			proofs, err := c.Proofs(agentstore.CapabilityQuery{
				Cmd: "/widget/crank",
				Sub: did.Undef,
			})
			require.NoError(t, err)
			require.ElementsMatch(t, []ucan.Delegation{powerlineDel}, proofs,
				"should only match powerline when query subject is empty")
		})
	})
}

func TestWithAdditionalProofs(t *testing.T) {
	t.Run("includes additional proofs in Proofs() results", func(t *testing.T) {
		// Create a store to verify what gets saved
		store := testutil.Must(agentstore.NewMemory())(t)

		// Create delegations
		s := testutil.RandomSigner(t)
		storedDel := testutil.Must(blob.Add.Delegate(s, s.DID(), s.DID()))(t)
		additionalDel := testutil.Must(blob.Add.Delegate(s, s.DID(), s.DID()))(t)

		// Create client with store and additional proofs
		c := testutil.Must(client.NewClient(
			client.WithStore(store),
			client.WithPrincipal(s),
			client.WithAdditionalProofs(additionalDel),
		))(t)

		// Add a proof to the store
		err := c.AddProofs(storedDel)
		require.NoError(t, err)

		// Verify that only the stored delegation was saved
		storedDelegations, err := store.Delegations()
		require.NoError(t, err)
		require.Equal(t, []ucan.Delegation{storedDel}, storedDelegations,
			"only stored delegation should be saved to storage")

		// Verify that Proofs() returns both stored and additional proofs
		proofs, err := c.Proofs()
		require.NoError(t, err)
		require.ElementsMatch(t, []ucan.Delegation{storedDel, additionalDel}, proofs,
			"Proofs() should return both stored and additional proofs")
	})

	t.Run("additional proofs respect filtering", func(t *testing.T) {
		s := testutil.RandomSigner(t)
		addDel := testutil.Must(blob.Add.Delegate(s, s.DID(), s.DID()))(t)
		listDel := testutil.Must(blob.List.Delegate(s, s.DID(), s.DID()))(t)

		// Create client with both delegations as additional proofs
		c := testutil.Must(client.NewClient(
			client.WithPrincipal(s),
			client.WithAdditionalProofs(addDel, listDel),
		))(t)

		// Query for only upload capabilities
		proofs, err := c.Proofs(agentstore.CapabilityQuery{
			Cmd: ucan.Command(blob.Add),
			Sub: s.DID(),
		})
		require.NoError(t, err)
		require.ElementsMatch(t, []ucan.Delegation{addDel}, proofs,
			"should filter additional proofs by capability query")
	})

	t.Run("additional proofs exclude expired delegations", func(t *testing.T) {
		s := testutil.RandomSigner(t)
		validDel := testutil.Must(blob.Add.Delegate(s, s.DID(), s.DID()))(t)
		expiredDel := testutil.Must(blob.Add.Delegate(s, s.DID(), s.DID(),
			delegation.WithExpiration(ucan.Now()-100), // Expired 100 seconds ago
		))(t)

		// Create client with both delegations as additional proofs
		c := testutil.Must(client.NewClient(
			client.WithPrincipal(s),
			client.WithAdditionalProofs(validDel, expiredDel),
		))(t)

		// Only the valid delegation should be returned
		proofs, err := c.Proofs()
		require.NoError(t, err)
		require.ElementsMatch(t, []ucan.Delegation{validDel}, proofs,
			"should exclude expired additional proofs")
	})

	t.Run("Reset does not affect additional proofs", func(t *testing.T) {
		store := testutil.Must(agentstore.NewMemory())(t)

		s := testutil.RandomSigner(t)
		storedDel := testutil.Must(blob.Add.Delegate(s, s.DID(), s.DID()))(t)
		additionalDel := testutil.Must(blob.Add.Delegate(s, s.DID(), s.DID()))(t)
		// Create client with additional proofs

		c := testutil.Must(client.NewClient(
			client.WithStore(store),
			client.WithPrincipal(s),
			client.WithAdditionalProofs(additionalDel),
		))(t)

		// Add a stored proof
		err := c.AddProofs(storedDel)
		require.NoError(t, err)

		// Verify both are returned
		proofs, err := c.Proofs()
		require.NoError(t, err)
		require.ElementsMatch(t, []ucan.Delegation{storedDel, additionalDel}, proofs)

		// Reset the client
		err = c.Reset()
		require.NoError(t, err)

		// Additional proofs should still be there, but stored proof should be gone
		proofs, err = c.Proofs()
		require.NoError(t, err)
		require.ElementsMatch(t, []ucan.Delegation{additionalDel}, proofs,
			"additional proofs should remain after reset, but stored proofs should be cleared")

		// Verify storage was cleared
		storedDelegations, err := store.Delegations()
		require.NoError(t, err)
		require.Empty(t, storedDelegations,
			"stored delegations should be cleared after reset")
	})
}
