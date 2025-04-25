package client_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/fluent/qp"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/multiformats/go-multihash"
	"github.com/storacha/go-libstoracha/capabilities/assert"
	blobcap "github.com/storacha/go-libstoracha/capabilities/blob"
	httpcap "github.com/storacha/go-libstoracha/capabilities/http"
	spaceblobcap "github.com/storacha/go-libstoracha/capabilities/space/blob"
	"github.com/storacha/go-libstoracha/capabilities/types"
	ucancap "github.com/storacha/go-libstoracha/capabilities/ucan"
	uploadcap "github.com/storacha/go-libstoracha/capabilities/upload"
	uclient "github.com/storacha/go-ucanto/client"
	"github.com/storacha/go-ucanto/core/delegation"
	"github.com/storacha/go-ucanto/core/invocation"
	"github.com/storacha/go-ucanto/core/invocation/ran"
	"github.com/storacha/go-ucanto/core/message"
	"github.com/storacha/go-ucanto/core/receipt"
	"github.com/storacha/go-ucanto/core/receipt/fx"
	"github.com/storacha/go-ucanto/core/result"
	"github.com/storacha/go-ucanto/core/result/failure"
	"github.com/storacha/go-ucanto/did"
	"github.com/storacha/go-ucanto/principal"
	ed25519signer "github.com/storacha/go-ucanto/principal/ed25519/signer"
	"github.com/storacha/go-ucanto/principal/signer"
	"github.com/storacha/go-ucanto/server"
	"github.com/storacha/go-ucanto/transport/car"
	carresp "github.com/storacha/go-ucanto/transport/car/response"
	uhttp "github.com/storacha/go-ucanto/transport/http"
	"github.com/storacha/go-ucanto/ucan"
	"github.com/stretchr/testify/require"

	"github.com/storacha/go-w3up/client"
)

var acceptInvocation invocation.Invocation

func TestBlobAdd(t *testing.T) {
	issuer, err := ed25519signer.Generate()
	if err != nil {
		t.Fatal(err)
	}

	space, err := ed25519signer.Generate()
	if err != nil {
		t.Fatal(err)
	}

	serviceKey, err := ed25519signer.Generate()
	if err != nil {
		t.Fatal(err)
	}

	servicePrincipal, err := did.Parse("did:web:storacha.test")
	if err != nil {
		t.Fatal(err)
	}

	serviceSigner, err := signer.Wrap(serviceKey, servicePrincipal)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	httpSrv := httptest.NewServer(mux)
	defer httpSrv.Close()

	ucanURL, err := url.Parse(httpSrv.URL + "/ucan")
	if err != nil {
		t.Fatal(err)
	}
	putBlobURL, err := url.Parse(httpSrv.URL + "/blob")
	if err != nil {
		t.Fatal(err)
	}
	receiptsURL, err := url.Parse(httpSrv.URL + "/receipt")
	if err != nil {
		t.Fatal(err)
	}

	ucanSrv := setupTestUCANServer(t, serviceSigner, putBlobURL)

	setupHTTPHandlers(t, mux, ucanSrv, ucanURL, putBlobURL, receiptsURL)

	channel := uhttp.NewHTTPChannel(ucanURL)
	codec := car.NewCAROutboundCodec()

	conn, err := uclient.NewConnection(servicePrincipal, channel, uclient.WithOutboundCodec(codec))
	if err != nil {
		t.Fatal(err)
	}

	// delegate * to the space
	cap := ucan.NewCapability("*", space.DID().String(), ucan.NoCaveats{})
	proof, err := delegation.Delegate(space, issuer, []ucan.Capability[ucan.NoCaveats]{cap}, delegation.WithNoExpiration())
	if err != nil {
		t.Fatal(err)
	}

	testBlob := bytes.NewReader([]byte("test"))

	_, _, err = client.BlobAdd(testBlob, issuer, space.DID(), receiptsURL, client.WithConnection(conn), client.WithProof(proof))
	require.NoError(t, err)
}

type httpPutFact struct {
	id  string
	key []byte
}

func (hpf httpPutFact) ToIPLD() (map[string]datamodel.Node, error) {
	n, err := qp.BuildMap(basicnode.Prototype.Any, 2, func(ma datamodel.MapAssembler) {
		qp.MapEntry(ma, "id", qp.String(hpf.id))
		qp.MapEntry(ma, "keys", qp.Map(2, func(ma datamodel.MapAssembler) {
			qp.MapEntry(ma, hpf.id, qp.Bytes(hpf.key))
		}))
	})
	if err != nil {
		return nil, err
	}

	return map[string]datamodel.Node{
		"keys": n,
	}, nil
}

func setupTestUCANServer(t *testing.T, serverPrincipal principal.Signer, putBlobURL *url.URL) server.ServerView {
	// space/blob/add handler
	mockSPKey, err := ed25519signer.Generate()
	if err != nil {
		t.Fatal(err)
	}

	mockSPDID, err := did.Parse("did:web:sp.test")
	if err != nil {
		t.Fatal(err)
	}

	mockSPPrincipal, err := signer.Wrap(mockSPKey, mockSPDID)
	if err != nil {
		t.Fatal(err)
	}

	spaceBlobAddMethod := server.Provide(
		spaceblobcap.Add,
		func(cap ucan.Capability[spaceblobcap.AddCaveats], inv invocation.Invocation, ctx server.InvocationContext) (spaceblobcap.AddOk, fx.Effects, error) {
			// add task for blob/allocate
			blobDigest := cap.Nb().Blob.Digest
			blobSize := cap.Nb().Blob.Size

			spaceDID, err := did.Parse(cap.With())
			if err != nil {
				t.Fatal(err)
			}
			allocateCaveats := blobcap.AllocateCaveats{
				Space: spaceDID,
				Blob: blobcap.Blob{
					Digest: blobDigest,
					Size:   blobSize,
				},
				Cause: inv.Link(),
			}
			allocateInv, err := blobcap.Allocate.Invoke(serverPrincipal, mockSPPrincipal, cap.With(), allocateCaveats)
			if err != nil {
				t.Fatal(err)
			}
			allocateInv.Attach(inv.Root())

			// add task for ucan/conclude (with the receipt of blob/allocate)
			allocateResult := result.Ok[blobcap.AllocateOk, failure.IPLDBuilderFailure](blobcap.AllocateOk{
				Size: blobSize,
				Address: &blobcap.Address{
					URL:     *putBlobURL,
					Headers: http.Header{"some-header": []string{"some-value"}},
					Expires: uint64(time.Now().Add(1 * time.Minute).Unix()),
				},
			})
			allocateRcpt, err := receipt.Issue(mockSPPrincipal, allocateResult, ran.FromInvocation(allocateInv))
			if err != nil {
				t.Fatal(err)
			}
			concludeCaveats := ucancap.ConcludeCaveats{
				Receipt: allocateRcpt.Root().Link(),
			}
			concludeInv, err := ucancap.Conclude.Invoke(serverPrincipal, mockSPPrincipal, cap.With(), concludeCaveats)
			if err != nil {
				t.Fatal(err)
			}
			concludeInv.Attach(allocateRcpt.Root())

			// add task for http/put
			httpPutCaveats := httpcap.PutCaveats{
				URL: types.Promise{
					UcanAwait: types.Await{
						Selector: ".out.ok.address.url",
						Link:     allocateRcpt.Root().Link()},
				},
				Headers: types.Promise{
					UcanAwait: types.Await{
						Selector: ".out.ok.address.headers",
						Link:     allocateRcpt.Root().Link()},
				},
				Body: httpcap.Body{
					Digest: blobDigest,
					Size:   blobSize,
				},
			}

			// random signer rather than the proper derived one
			//blobProvider, err := ed25519signer.FromSeed([]byte(blobDigest)[len(blobDigest)-32:])
			blobProvider, err := ed25519signer.Generate()
			if err != nil {
				t.Fatal(err)
			}

			fct := httpPutFact{
				id:  blobProvider.DID().String(),
				key: blobProvider.Encode(),
			}

			facts := []ucan.FactBuilder{fct}
			httpPutInv, err := httpcap.Put.Invoke(blobProvider, blobProvider, blobProvider.DID().String(), httpPutCaveats, delegation.WithFacts(facts))
			if err != nil {
				t.Fatal(err)
			}
			httpPutInv.Attach(allocateRcpt.Root())

			// add task for blob/accept
			acceptCaveats := blobcap.AcceptCaveats{
				Space: spaceDID,
				Blob: blobcap.Blob{
					Digest: blobDigest,
					Size:   blobSize,
				},
				Put: blobcap.Promise{
					UcanAwait: blobcap.Await{
						Selector: ".out.ok",
						Link:     httpPutInv.Root().Link(),
					},
				},
			}
			acceptInv, err := blobcap.Accept.Invoke(serverPrincipal, mockSPPrincipal, mockSPPrincipal.DID().String(), acceptCaveats)
			if err != nil {
				t.Fatal(err)
			}
			acceptInv.Attach(httpPutInv.Root())

			acceptInvocation = acceptInv

			forks := []fx.Effect{
				fx.FromInvocation(allocateInv),
				fx.FromInvocation(concludeInv),
				fx.FromInvocation(httpPutInv),
				fx.FromInvocation(acceptInv),
			}
			fxs := fx.NewEffects(fx.WithFork(forks...))

			ok := spaceblobcap.AddOk{
				Site: spaceblobcap.Promise{
					UcanAwait: spaceblobcap.Await{
						Selector: ".out.ok.site",
						Link:     acceptInv.Root().Link(),
					},
				},
			}

			return ok, fxs, nil
		},
	)

	// ucan/conclude handler
	ucanConcludeMethod := server.Provide(
		ucancap.Conclude,
		func(capability ucan.Capability[ucancap.ConcludeCaveats], invocation invocation.Invocation, context server.InvocationContext) (ucancap.ConcludeOk, fx.Effects, error) {
			return ucancap.ConcludeOk{}, nil, nil
		},
	)

	// upload/add handler
	uploadAddMethod := server.Provide(
		uploadcap.Add,
		func(capability ucan.Capability[uploadcap.AddCaveats], invocation invocation.Invocation, context server.InvocationContext) (uploadcap.AddOk, fx.Effects, error) {
			return uploadcap.AddOk{}, nil, nil
		},
	)

	srv, err := server.NewServer(
		serverPrincipal,
		server.WithInboundCodec(car.NewCARInboundCodec()),
		server.WithServiceMethod(spaceblobcap.AddAbility, spaceBlobAddMethod),
		server.WithServiceMethod(ucancap.ConcludeAbility, ucanConcludeMethod),
		server.WithServiceMethod(uploadcap.AddAbility, uploadAddMethod),
	)
	if err != nil {
		t.Fatal(err)
	}

	return srv
}

func setupHTTPHandlers(t *testing.T, mux *http.ServeMux, ucanSrv server.ServerView, ucanURL, putBlobURL, receiptsURL *url.URL) {
	// ucan handler
	ucanPath := fmt.Sprintf("POST %s", ucanURL.Path)
	mux.HandleFunc(ucanPath, func(w http.ResponseWriter, r *http.Request) {
		res, _ := ucanSrv.Request(uhttp.NewHTTPRequest(r.Body, r.Header))

		for key, vals := range res.Headers() {
			for _, v := range vals {
				w.Header().Add(key, v)
			}
		}

		if res.Status() != 0 {
			w.WriteHeader(res.Status())
		}

		_, err := io.Copy(w, res.Body())
		if err != nil {
			t.Fatal(err)
		}
	})

	// put blob handler
	putBlobPath := fmt.Sprintf("PUT %s", putBlobURL.Path)
	mux.HandleFunc(putBlobPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// receipts handler
	receiptsPath := fmt.Sprintf("GET %s/{cid}", receiptsURL.Path)
	mux.HandleFunc(receiptsPath, func(w http.ResponseWriter, r *http.Request) {
		cidStr := r.PathValue("cid")
		if cidStr == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_, err := cid.Parse(cidStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		issuer, err := ed25519signer.Generate()
		if err != nil {
			t.Fatal(err)
		}

		space, err := ed25519signer.Generate()
		if err != nil {
			t.Fatal(err)
		}

		// not a valid location claim, but enough for testing
		locationClaim, err := assert.Location.Delegate(
			issuer,
			space.DID(),
			space.DID().String(),
			assert.LocationCaveats{
				Space:    space.DID(),
				Content:  types.FromHash(randomMultihash(t)),
				Location: []url.URL{*putBlobURL},
			},
			delegation.WithNoExpiration(),
		)
		if err != nil {
			t.Fatal(err)
		}

		ok := result.Ok[blobcap.AcceptOk, failure.IPLDBuilderFailure](blobcap.AcceptOk{
			Site: locationClaim.Link(),
		})

		forks := []fx.Effect{fx.FromInvocation(locationClaim)}

		acceptRcpt, err := receipt.Issue(issuer, ok, ran.FromInvocation(acceptInvocation), receipt.WithFork(forks...))
		if err != nil {
			t.Fatal(err)
		}

		msg, err := message.Build(nil, []receipt.AnyReceipt{acceptRcpt})
		if err != nil {
			t.Fatal(err)
		}

		resp, err := carresp.Encode(msg)
		if err != nil {
			t.Fatal(err)
		}

		for hdrK, hdrV := range resp.Headers() {
			w.Header().Add(hdrK, hdrV[0])
		}

		_, err = io.Copy(w, resp.Body())
		if err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(resp.Status())
	})
}

func randomMultihash(t *testing.T) multihash.Multihash {
	bytes := make([]byte, 10)
	_, err := rand.Read(bytes)
	if err != nil {
		t.Fatal(err)
	}

	digest, err := multihash.Sum(bytes, multihash.SHA2_256, -1)
	if err != nil {
		t.Fatal(err)
	}

	return digest
}
