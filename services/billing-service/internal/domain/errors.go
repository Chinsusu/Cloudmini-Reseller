package domain

import "errors"

var (
	ErrWalletNotFound       = errors.New("wallet not found")
	ErrInsufficientFunds    = errors.New("insufficient funds")
	ErrInvalidAmount        = errors.New("amount must be positive")
	ErrPaymentNotFound      = errors.New("payment not found")
	ErrPaymentAlreadyExists = errors.New("payment already processed")
	ErrUnsupportedGateway   = errors.New("unsupported payment gateway")
	ErrPricingRuleNotFound  = errors.New("pricing rule not found")
)
