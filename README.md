# go-w3up

A w3up client in golang. ⚠️ Heavily WIP.

## Install

```sh
go get github.com/web3-storage/go-w3up
```

## Usage

## Client library

Example listing uploads:

```go
package main

import (
  "net/url"
  "ioutil"

  "github.com/web3-storage/go-ucanto/core/delegation"
  "github.com/web3-storage/go-ucanto/did"
  "github.com/web3-storage/go-ucanto/principal/ed25519/signer"
  "github.com/web3-storage/go-w3up/client"
)

// private key to sign invocation UCAN with
priv, _ := ioutil.ReadFile("path/to/private.key")
signer, _ := signer.Parse(priv)

// UCAN proof that signer can list uploads in this space
prfbytes, _ := ioutil.ReadFile("path/to/proof.ucan")
proof, _ := delegation.Extract(b)

// space to list uploads from
space, _ := did.Parse("did:key:z6MkwDuRThQcyWjqNsK54yKAmzfsiH6BTkASyiucThMtHt1y")

rcpt, _ := client.UploadList(
   signer,
   space,
   &uploadlist.Caveat{},
   client.WithProofs([]delegation.Delegation{proof}),
)

for _, r := range rcpt.Out().Ok().Results {
   fmt.Printf("%s\n", r.Root)
}
```

## CLI

```console
go run ./cmd/w3.go --help
NAME:
   w3 - interact with the web3.storage API

USAGE:
   w3 [global options] command [command options] [arguments...]

COMMANDS:
   whoami      Print information about the current agent.
   up, upload  Store a file(s) to the service and register an upload.
   ls, list    List uploads in the current space.
   help, h     Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help
```

## Contributing

Feel free to join in. All welcome. Please [open an issue](https://github.com/web3-storage/go-w3up/issues)!

## License

Dual-licensed under [MIT + Apache 2.0](LICENSE.md)
