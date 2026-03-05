package domain

import "errors"

var (
	ErrProviderNotFound    = errors.New("provider not found")
	ErrProductNotFound     = errors.New("product not found")
	ErrOrderNotFound       = errors.New("order not found")
	ErrOrderAlreadyExists  = errors.New("order with this idempotency key already exists")
	ErrNoProviderAvailable = errors.New("no provider available for this product")
	ErrProviderPurchase    = errors.New("provider purchase failed")
	ErrCredentialEncrypt   = errors.New("failed to encrypt credentials")
	ErrOrderNotCancellable = errors.New("order cannot be cancelled in current status")
)
