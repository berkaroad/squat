package tests

import (
	"fmt"

	"github.com/berkaroad/squat/domain"
)

func CreateAccount(existsAccount *Account, accountInfo AccountInfo) (*Account, error) {
	result := existsAccount
	if existsAccount == nil {
		result = &Account{}
	} else if !existsAccount.state.IsDeleted || existsAccount.AggregateID() != accountInfo.ID {
		return nil, fmt.Errorf("%w: account-id=%s", ErrAccountAlreadyExists, existsAccount.AggregateID())
	}
	result.Apply(&AccountCreated{
		DomainEventBase: domain.NewDomainEventBase(NewUUID()),
		ID:              accountInfo.ID,
		Name:            accountInfo.Name,
		Balance:         accountInfo.Balance,
	})
	return result, nil
}

func (a *Account) Deposit(amount float64) error {
	if a == nil {
		return ErrAccounteNotExists
	}
	if a.state.IsDeleted {
		return ErrAccountHasDeleted
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}
	a.Apply(&AccountBalanceChanged{
		DomainEventBase: domain.NewDomainEventBase(NewUUID()),
		Amount:          amount,
		Source:          BalanceChangedSourceDeposit,
	})
	return nil
}

func (a *Account) WithDraw(amount float64) error {
	if a == nil {
		return ErrAccounteNotExists
	}
	if a.state.IsDeleted {
		return ErrAccountHasDeleted
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if amount > a.state.Balance {
		return ErrInsufficientBalance
	}
	a.Apply(&AccountBalanceChanged{
		DomainEventBase: domain.NewDomainEventBase(NewUUID()),
		Amount:          -amount,
		Source:          BalanceChangedSourceWithdraw,
	})
	return nil
}

func (a *Account) Remove() error {
	if a == nil || a.state.IsDeleted {
		return nil
	}
	if a.state.Balance != 0 {
		return ErrAccountHasBalance
	}
	a.Apply(&AccountRemoved{
		DomainEventBase: domain.NewDomainEventBase(NewUUID()),
	})
	return nil
}
