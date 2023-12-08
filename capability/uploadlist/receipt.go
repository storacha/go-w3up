package uploadlist

import "github.com/web3-storage/go-ucanto/core/receipt"

func NewReceiptReader() (receipt.ReceiptReader[*UploadListSuccess, *UploadListFailure], error) {
	return receipt.NewReceiptReader[*UploadListSuccess, *UploadListFailure](UploadSchema)
}
