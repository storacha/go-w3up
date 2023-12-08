package storeadd

import "github.com/web3-storage/go-ucanto/core/receipt"

func NewReceiptReader() (receipt.ReceiptReader[*Success, *Failure], error) {
	return receipt.NewReceiptReader[*Success, *Failure](ResultSchema)
}
