package tests

import "github.com/berkaroad/squat/errors"

const (
	ErrCodeAccountAlreadyExists string = "AccountAlreadyExists"
	ErrCodeAccountNotExists     string = "AccountNotExists"
	ErrCodeAccountHasDeleted    string = "AccountHasDeleted"
	ErrCodeInvalidAmount        string = "InvalidAmount"
	ErrCodeInsufficientBalance  string = "InsufficientBalance"
	ErrCodeAccountHasBalance    string = "AccountHasBalance"
)

var (
	ErrAccountAlreadyExists error = errors.NewWithCode(ErrCodeAccountAlreadyExists, "account already exists")
	ErrAccounteNotExists    error = errors.NewWithCode(ErrCodeAccountNotExists, "account not exists")
	ErrAccountHasDeleted    error = errors.NewWithCode(ErrCodeAccountHasDeleted, "account has deleted")
	ErrInvalidAmount        error = errors.NewWithCode(ErrCodeInvalidAmount, "invalid amount")
	ErrInsufficientBalance  error = errors.NewWithCode(ErrCodeInsufficientBalance, "insufficient balance")
	ErrAccountHasBalance    error = errors.NewWithCode(ErrCodeAccountHasBalance, "account has balance")
)
