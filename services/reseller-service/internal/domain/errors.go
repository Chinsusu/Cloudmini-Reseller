package domain

import "errors"

var (
	ErrResellerNotFound      = errors.New("reseller not found")
	ErrResellerNotApproved   = errors.New("reseller account is not approved")
	ErrResellerSuspended     = errors.New("reseller account is suspended")
	ErrResellerAlreadyExists = errors.New("reseller account already exists for this user")
	ErrAPIKeyNotFound        = errors.New("api key not found")
	ErrAPIKeyRevoked         = errors.New("api key has been revoked")
	ErrAPIKeyExpired         = errors.New("api key has expired")
	ErrSubAccountNotFound    = errors.New("sub-account not found")
	ErrPricingNotFound       = errors.New("pricing override not found")
	ErrPriceBelowCost        = errors.New("sell price cannot be below cost price")
	ErrPriceBelowFloor       = errors.New("sell price cannot be below floor price")
	ErrPriceAboveCap         = errors.New("sell price exceeds maximum allowed (100x cost)")
	ErrWebhookNotFound       = errors.New("webhook not found")
	ErrForbidden             = errors.New("forbidden")
)
