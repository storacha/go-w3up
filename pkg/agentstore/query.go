package agentstore

import (
	"github.com/fil-forge/ucantone/ucan"
)

// Query returns delegations that match the given capability queries.
// If no queries are provided, returns all non-expired delegations.
// Delegations are filtered by:
//   - Expiration: excludes expired delegations
//   - NotBefore: excludes delegations that are not yet valid
//   - Capability matching: if queries are provided, only returns delegations
//     whose capabilities match at least one of the queries
//
// Additionally, this method includes relevant session proofs (ucan/attest delegations)
// that attest to the returned authorizations.
func Query(delegations []ucan.Delegation, queries []CapabilityQuery) []ucan.Delegation {
	panic("not reimplemented")

	// now := ucan.Now()

	// // Map to track which delegations to include (by CID string)
	// authorizations := make(map[string]ucan.Delegation)

	// // First pass: collect matching authorizations (non-session-proof delegations)
	// for _, del := range delegations {
	// 	// Filter out expired delegations
	// 	if exp := del.Expiration(); exp != nil && *exp < now {
	// 		continue
	// 	}

	// 	// Filter out delegations that are not yet valid
	// 	if nb := del.NotBefore(); nb != nil && *nb > now {
	// 		continue
	// 	}

	// 	// Skip session proofs in the first pass
	// 	// if isSessionProof(del) {
	// 	// 	continue
	// 	// }

	// 	// If no queries, include all non-expired delegations
	// 	if len(queries) == 0 {
	// 		authorizations[del.Link().String()] = del
	// 		continue
	// 	}

	// 	// Check if delegation matches any query
	// 	if matchesAnyQuery(del, queries) {
	// 		authorizations[del.Link().String()] = del
	// 	}
	// }

	// // Second pass: collect session proofs that attest to the authorizations
	// // sessionProofs := getSessionProofs(delegations, now)

	// // for authCID := range authorizations {
	// // 	if proofsForAuth, exists := sessionProofs[authCID]; exists {
	// // 		// Add all session proofs for this authorization
	// // 		for _, sessionProof := range proofsForAuth {
	// // 			authorizations[sessionProof.Link().String()] = sessionProof
	// // 		}
	// // 	}
	// // }

	// // Convert map to slice
	// result := make([]ucan.Delegation, 0, len(authorizations))
	// for _, del := range authorizations {
	// 	result = append(result, del)
	// }

	// return result

}

// // matchesAnyQuery checks if a delegation's capabilities match any of the provided queries.
// func matchesAnyQuery(del ucan.Delegation, queries []CapabilityQuery) bool {
// 	for _, query := range queries {
// 		if matchesDelegation(del, query) {
// 			return true
// 		}
// 	}
// 	return false
// }

// // matchesDelegation checks if a capability matches a query using the resolution logic
// // from go-ucanto's validator package.
// func matchesDelegation(del ucan.Delegation, query CapabilityQuery) bool {
// 	// Match ability
// 	if !matchesAbility(del.Can(), query.Can) {
// 		return false
// 	}

// 	// Match resource
// 	if !matchesSubject(del.With(), query.Sub) {
// 		return false
// 	}

// 	return true
// }

// // matchesAbility checks if a delegation's ability can authorize the query ability.
// // A delegation matches if it has the same ability or a broader wildcard.
// // For example, searching for "upload/add" matches delegations with:
// //   - "upload/add" (exact match)
// //   - "upload/*" (namespace wildcard)
// //   - "*" (global wildcard)
// func matchesAbility(capAbility, queryAbility ucan.Ability) bool {
// 	// Exact match
// 	if capAbility == queryAbility {
// 		return true
// 	}

// 	// Global wildcard in capability
// 	if capAbility == "*" {
// 		return true
// 	}

// 	// Namespace wildcard in capability (e.g., capability has "upload/*", query is "upload/add")
// 	if len(capAbility) > 2 && capAbility[len(capAbility)-2:] == "/*" {
// 		prefix := capAbility[:len(capAbility)-1] // "upload/"
// 		return len(queryAbility) >= len(prefix) && queryAbility[:len(prefix)] == prefix
// 	}

// 	return false
// }

// // TK: Make sure this still makes sense.
// // matchesSubject checks if a delegation's subject matches the query subject.
// // A delegation matches if:
// //   - The query subject is empty (matches any delegation subject)
// //   - The delegation is empty (matches any query subject)
// //   - The subjects match exactly
// func matchesSubject(capResource, queryResource ucan.Resource) bool {
// 	return queryResource == "" || capResource == "ucan:*" || queryResource == capResource
// }

// // isSessionProof checks if a delegation is a session proof (/ucan/attest/proof command).
// func isSessionProof(del ucan.Delegation) bool {
// 	return del.Command() == ucan.Command(attest.Proof)
// }

// // getSessionProofs organizes session proofs by the CID of the authorization they attest to.
// // Returns a map from authorization CID string to list of session proof delegations.
// func getSessionProofs(delegations []ucan.Delegation, now ucan.UnixTimestamp) map[string][]ucan.Delegation {
// 	proofs := make(map[string][]ucan.Delegation)

// 	for _, del := range delegations {
// 		if !isSessionProof(del) {
// 			continue
// 		}

// 		// Filter out expired session proofs
// 		if exp := del.Expiration(); exp != nil && *exp < now {
// 			continue
// 		}

// 		// Filter out session proofs that are not yet valid
// 		if nb := del.NotBefore(); nb != nil && *nb > now {
// 			continue
// 		}

// 		// Get the proof link from the typed capability
// 		attestCap := match.Value()
// 		proofCID := attestCap.Nb().Proof.String()
// 		proofs[proofCID] = append(proofs[proofCID], del)
// 	}

// 	return proofs
// }
