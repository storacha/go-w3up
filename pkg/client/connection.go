package client

import (
	"net/url"

	ucantoclient "github.com/fil-forge/ucantone/client"
	"github.com/fil-forge/ucantone/execution"
	"github.com/storacha/guppy/pkg/receipt"
)

type UcantoneClient interface {
	Execute(execRequest execution.Request) (execution.Response, error)
}

var DefaultUcantoneClient UcantoneClient
var DefaultReceiptsClient *receipt.Client

func init() {
	// service URL & DID
	serviceURL, err := url.Parse("https://up.web3.storage")
	if err != nil {
		log.Fatal(err)
	}

	// servicePrincipal, err := did.Parse("did:web:web3.storage")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// // HTTP transport and CAR encoding
	// channel := http.NewChannel(serviceURL)
	// codec := car.NewOutboundCodec()

	DefaultUcantoneClient, err = ucantoclient.NewHTTP(serviceURL)
	if err != nil {
		log.Fatal(err)
	}

	defaultReceiptsURL := serviceURL.JoinPath("receipt")
	DefaultReceiptsClient = receipt.New(defaultReceiptsURL)
}
