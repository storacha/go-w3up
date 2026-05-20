package agentstore

import (
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"

	"github.com/fil-forge/libforge/commands/blob"
	"github.com/fil-forge/libforge/commands/upload"
	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/testutil"
	"github.com/fil-forge/ucantone/ucan"
	"github.com/fil-forge/ucantone/ucan/delegation"
)

// TK: Still valid?
// delegationLinks extracts the CID links from a slice of delegations for comparison.
// This is necessary because delegations may not compare equal after serialization/deserialization.
func delegationLinks(dels []ucan.Delegation) []cid.Cid {
	links := make([]cid.Cid, len(dels))
	for i, d := range dels {
		links[i] = d.Link()
	}
	return links
}

func TestStore(t *testing.T) {
	stores := map[string]func(t *testing.T) Store{
		"MemStore": func(t *testing.T) Store {
			s, err := NewMemory()
			require.NoError(t, err)
			return s
		},
		"FsStore": func(t *testing.T) Store {
			s, err := NewFs(t.TempDir())
			require.NoError(t, err)
			return s
		},
	}

	for name, newStore := range stores {
		t.Run(name, func(t *testing.T) {
			t.Run("Principal", func(t *testing.T) { testPrincipal(t, newStore) })
			t.Run("Delegations", func(t *testing.T) { testDelegations(t, newStore) })
			t.Run("Query", func(t *testing.T) { testQuery(t, newStore) })
			t.Run("Reset", func(t *testing.T) { testReset(t, newStore) })
		})
	}
}

func testPrincipal(t *testing.T, newStore func(t *testing.T) Store) {
	t.Run("has principal after creation", func(t *testing.T) {
		s := newStore(t)
		has, err := s.HasPrincipal()
		require.NoError(t, err)
		require.True(t, has, "store should have a principal after creation")
	})

	t.Run("get principal", func(t *testing.T) {
		s := newStore(t)
		p, err := s.Principal()
		require.NoError(t, err)
		require.NotNil(t, p, "principal should not be nil")
		require.NotEmpty(t, p.DID().String(), "principal should have a DID")
	})

	t.Run("set principal", func(t *testing.T) {
		s := newStore(t)
		newPrincipal := testutil.RandomSigner(t)

		err := s.SetPrincipal(newPrincipal)
		require.NoError(t, err)

		p, err := s.Principal()
		require.NoError(t, err)
		require.Equal(t, newPrincipal.DID(), p.DID(), "principal should be updated")
	})
}

func testDelegations(t *testing.T, newStore func(t *testing.T) Store) {
	t.Run("empty delegations initially", func(t *testing.T) {
		s := newStore(t)
		delegs, err := s.Delegations()
		require.NoError(t, err)
		require.Empty(t, delegs, "delegations should be empty initially")
	})

	t.Run("add delegations", func(t *testing.T) {
		s := newStore(t)
		p, err := s.Principal()
		require.NoError(t, err)

		del := testutil.Must(upload.Add.Delegate(
			p, p.DID(), p.DID(),
		))(t)

		err = s.AddDelegations(del)
		require.NoError(t, err)

		delegs, err := s.Delegations()
		require.NoError(t, err)
		require.Len(t, delegs, 1, "should have one delegation")
		require.Equal(t, del.Link(), delegs[0].Link(), "delegation should match")
	})

	t.Run("add multiple delegations", func(t *testing.T) {
		s := newStore(t)
		p, err := s.Principal()
		require.NoError(t, err)

		del1 := testutil.Must(upload.Add.Delegate(
			p, p.DID(), p.DID(),
		))(t)

		del2 := testutil.Must(upload.List.Delegate(
			p, p.DID(), p.DID(),
		))(t)

		err = s.AddDelegations(del1, del2)
		require.NoError(t, err)

		delegs, err := s.Delegations()
		require.NoError(t, err)
		require.Len(t, delegs, 2, "should have two delegations")
	})
}

func testQuery(t *testing.T, newStore func(t *testing.T) Store) {
	t.Run("no query returns all non-expired delegations", func(t *testing.T) {
		s := newStore(t)
		p, err := s.Principal()
		require.NoError(t, err)

		uploadDel := testutil.Must(upload.Add.Delegate(
			p, p.DID(), p.DID(),
		))(t)

		blobDel := testutil.Must(blob.Add.Delegate(
			p, p.DID(), p.DID(),
		))(t)

		err = s.AddDelegations(uploadDel, blobDel)
		require.NoError(t, err)

		proofs, err := s.Query()
		require.NoError(t, err)
		require.ElementsMatch(t, delegationLinks([]ucan.Delegation{uploadDel, blobDel}), delegationLinks(proofs))
	})

	t.Run("query by specific ability", func(t *testing.T) {
		s := newStore(t)
		p, err := s.Principal()
		require.NoError(t, err)

		uploadDel := testutil.Must(upload.Add.Delegate(
			p, p.DID(), p.DID(),
		))(t)

		blobDel := testutil.Must(blob.Add.Delegate(
			p, p.DID(), p.DID(),
		))(t)

		err = s.AddDelegations(uploadDel, blobDel)
		require.NoError(t, err)

		proofs, err := s.Query(CapabilityQuery{
			Cmd: "/upload/add",
			Sub: p.DID(),
		})
		require.NoError(t, err)
		require.ElementsMatch(t, delegationLinks([]ucan.Delegation{uploadDel}), delegationLinks(proofs))
	})

	t.Run("query with empty resource matches any resource", func(t *testing.T) {
		s := newStore(t)
		p, err := s.Principal()
		require.NoError(t, err)

		del := testutil.Must(upload.Add.Delegate(
			p, p.DID(), p.DID(),
		))(t)

		err = s.AddDelegations(del)
		require.NoError(t, err)

		proofs, err := s.Query(CapabilityQuery{
			Cmd: "/upload/add",
			Sub: did.Undef, // should match any
		})
		require.NoError(t, err)
		require.ElementsMatch(t, delegationLinks([]ucan.Delegation{del}), delegationLinks(proofs))
	})

	t.Run("multiple queries", func(t *testing.T) {
		s := newStore(t)
		p, err := s.Principal()
		require.NoError(t, err)

		uploadDel := testutil.Must(upload.Add.Delegate(
			p, p.DID(), p.DID(),
		))(t)

		blobDel := testutil.Must(blob.Add.Delegate(
			p, p.DID(), p.DID(),
		))(t)

		err = s.AddDelegations(uploadDel, blobDel)
		require.NoError(t, err)

		proofs, err := s.Query(
			CapabilityQuery{Cmd: "/upload/add", Sub: p.DID()},
			CapabilityQuery{Cmd: "/blob/add", Sub: p.DID()},
		)
		require.NoError(t, err)
		require.ElementsMatch(t, delegationLinks([]ucan.Delegation{uploadDel, blobDel}), delegationLinks(proofs))
	})

	t.Run("non-matching query", func(t *testing.T) {
		s := newStore(t)
		p, err := s.Principal()
		require.NoError(t, err)

		del := testutil.Must(upload.Add.Delegate(
			p, p.DID(), p.DID(),
		))(t)

		err = s.AddDelegations(del)
		require.NoError(t, err)

		proofs, err := s.Query(CapabilityQuery{
			Cmd: "nonexistent/capability",
			Sub: did.Undef,
		})
		require.NoError(t, err)
		require.Empty(t, proofs)
	})

	t.Run("excludes expired delegations", func(t *testing.T) {
		s := newStore(t)
		p, err := s.Principal()
		require.NoError(t, err)

		validDel := testutil.Must(upload.Add.Delegate(
			p, p.DID(), p.DID(),
		))(t)

		expiredDel := testutil.Must(upload.List.Delegate(
			p, p.DID(), p.DID(),
			delegation.WithExpiration(ucan.Now()-100), // Expired 100 seconds ago
		))(t)

		err = s.AddDelegations(validDel, expiredDel)
		require.NoError(t, err)

		proofs, err := s.Query()
		require.NoError(t, err)
		require.ElementsMatch(t, delegationLinks([]ucan.Delegation{validDel}), delegationLinks(proofs))
	})

	t.Run("excludes future delegations", func(t *testing.T) {
		s := newStore(t)
		p, err := s.Principal()
		require.NoError(t, err)

		validDel := testutil.Must(upload.Add.Delegate(
			p, p.DID(), p.DID(),
		))(t)

		futureDel := testutil.Must(upload.List.Delegate(
			p, p.DID(), p.DID(),
			delegation.WithNotBefore(ucan.Now()+100), // Valid 100 seconds from now
		))(t)

		err = s.AddDelegations(validDel, futureDel)
		require.NoError(t, err)

		proofs, err := s.Query()
		require.NoError(t, err)
		require.ElementsMatch(t, delegationLinks([]ucan.Delegation{validDel}), delegationLinks(proofs))
	})

	// TK:
	// t.Run("session proofs", func(t *testing.T) {
	// 	t.Run("includes session proofs with authorization", func(t *testing.T) {
	// 		s := newStore(t)
	// 		p, err := s.Principal()
	// 		require.NoError(t, err)

	// 		issuer := testutil.RandomSigner(t)

	// 		authDel := testutil.Must(upload.Add.Delegate(
	// 			issuer, p.DID(), p.DID(),
	// 		))(t)

	// 		sessionProof := testutil.Must(attest.Proof.Delegate(
	// 			issuer, p.DID(), issuer.DID(),
	// 			attest.ProofCaveats{Proof: authDel.Link()},
	// 		))(t)

	// 		err = s.AddDelegations(authDel, sessionProof)
	// 		require.NoError(t, err)

	// 		proofs, err := s.Query()
	// 		require.NoError(t, err)
	// 		require.ElementsMatch(t, delegationLinks([]ucan.Delegation{authDel, sessionProof}), delegationLinks(proofs))
	// 	})

	// 	t.Run("includes session proofs when querying by capability", func(t *testing.T) {
	// 		s := newStore(t)
	// 		p, err := s.Principal()
	// 		require.NoError(t, err)

	// 		issuer := testutil.Must(signer.Generate())(t)

	// 		authDel := testutil.Must(upload.Add.Delegate(
	// 			issuer, p, p.DID().String(),
	// 		))(t)

	// 		sessionProof := testutil.Must(attest.Proof.Delegate(
	// 			issuer, p, issuer.DID().String(),
	// 			attest.ProofCaveats{Proof: authDel.Link()},
	// 		))(t)

	// 		err = s.AddDelegations(authDel, sessionProof)
	// 		require.NoError(t, err)

	// 		proofs, err := s.Query(CapabilityQuery{
	// 			Cmd: "/upload/add",
	// 			Sub: p.DID().String(),
	// 		})
	// 		require.NoError(t, err)
	// 		require.ElementsMatch(t, delegationLinks([]ucan.Delegation{authDel, sessionProof}), delegationLinks(proofs))
	// 	})

	// 	t.Run("excludes expired session proofs", func(t *testing.T) {
	// 		s := newStore(t)
	// 		p, err := s.Principal()
	// 		require.NoError(t, err)

	// 		issuer := testutil.Must(signer.Generate())(t)

	// 		authDel := testutil.Must(upload.Add.Delegate(
	// 			issuer, p, p.DID().String(),
	// 		))(t)

	// 		expiredSessionProof := testutil.Must(attest.Proof.Delegate(
	// 			issuer, p, issuer.DID().String(),
	// 			attest.ProofCaveats{Proof: authDel.Link()},
	// 			delegation.WithExpiration(ucan.Now()-100), // Expired
	// 		))(t)

	// 		err = s.AddDelegations(authDel, expiredSessionProof)
	// 		require.NoError(t, err)

	// 		proofs, err := s.Query()
	// 		require.NoError(t, err)
	// 		require.ElementsMatch(t, delegationLinks([]ucan.Delegation{authDel}), delegationLinks(proofs))
	// 	})
	// })

	t.Run("ability wildcard matching", func(t *testing.T) {
		s := newStore(t)
		p, err := s.Principal()
		require.NoError(t, err)

		specificDel, err := delegation.Delegate(p, p.DID(), did.Undef, "/upload/add")
		require.NoError(t, err)
		namespaceDel, err := delegation.Delegate(p, p.DID(), did.Undef, "/upload")
		require.NoError(t, err)
		topDel, err := delegation.Delegate(p, p.DID(), did.Undef, "/")
		require.NoError(t, err)
		err = s.AddDelegations(specificDel, namespaceDel, topDel)
		require.NoError(t, err)

		t.Run("specific query matches exact, namespace, and top", func(t *testing.T) {
			proofs, err := s.Query(CapabilityQuery{
				Cmd: "/upload/add",
				Sub: did.Undef,
			})
			require.NoError(t, err)
			require.ElementsMatch(t, delegationLinks([]ucan.Delegation{specificDel, namespaceDel, topDel}), delegationLinks(proofs))
		})

		t.Run("namespace query only matches namespace and top", func(t *testing.T) {
			proofs, err := s.Query(CapabilityQuery{
				Cmd: "/upload",
				Sub: did.Undef,
			})
			require.NoError(t, err)
			require.ElementsMatch(t, delegationLinks([]ucan.Delegation{namespaceDel, topDel}), delegationLinks(proofs))
		})

		t.Run("top query only matches top delegation", func(t *testing.T) {
			proofs, err := s.Query(CapabilityQuery{
				Cmd: "/",
				Sub: did.Undef,
			})
			require.NoError(t, err)
			require.ElementsMatch(t, delegationLinks([]ucan.Delegation{topDel}), delegationLinks(proofs))
		})
	})

	t.Run("powerline matching", func(t *testing.T) {
		s := newStore(t)
		p, err := s.Principal()
		require.NoError(t, err)
		space := testutil.RandomSigner(t)

		specificResourceDel := testutil.Must(upload.Add.Delegate(
			p, p.DID(), space.DID(),
		))(t)

		powerlineResourceDel, err := delegation.Delegate(p, p.DID(), did.Undef, "/upload/add")
		require.NoError(t, err)

		err = s.AddDelegations(specificResourceDel, powerlineResourceDel)
		require.NoError(t, err)

		t.Run("specific subject query matches subject and powerline", func(t *testing.T) {
			proofs, err := s.Query(CapabilityQuery{
				Cmd: "/upload/add",
				Sub: space.DID(),
			})
			require.NoError(t, err)
			require.ElementsMatch(t, delegationLinks([]ucan.Delegation{specificResourceDel, powerlineResourceDel}), delegationLinks(proofs))
		})

		t.Run("powerline query only matches powerline", func(t *testing.T) {
			proofs, err := s.Query(CapabilityQuery{
				Cmd: "/upload/add",
				Sub: did.Undef,
			})
			require.NoError(t, err)
			require.ElementsMatch(t, delegationLinks([]ucan.Delegation{powerlineResourceDel}), delegationLinks(proofs))
		})
	})
}

func testReset(t *testing.T, newStore func(t *testing.T) Store) {
	t.Run("clears delegations but preserves principal", func(t *testing.T) {
		s := newStore(t)
		p, err := s.Principal()
		require.NoError(t, err)

		del := testutil.Must(upload.Add.Delegate(
			p, p.DID(), p.DID(),
		))(t)

		err = s.AddDelegations(del)
		require.NoError(t, err)

		delegs, err := s.Delegations()
		require.NoError(t, err)
		require.Len(t, delegs, 1)

		err = s.Reset()
		require.NoError(t, err)

		delegs, err = s.Delegations()
		require.NoError(t, err)
		require.Empty(t, delegs, "delegations should be empty after reset")

		newP, err := s.Principal()
		require.NoError(t, err)
		require.Equal(t, p.DID(), newP.DID(), "principal should be preserved after reset")
	})
}
