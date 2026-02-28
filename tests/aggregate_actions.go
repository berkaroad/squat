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

func (d *Account) Deposit(amount float64) error {
	if d == nil {
		return ErrAccounteNotExists
	}
	if d.state.IsDeleted {
		return ErrAccountHasDeleted
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}
	d.Apply(&AccountBalanceChanged{
		DomainEventBase: domain.NewDomainEventBase(NewUUID()),
		Amount:          amount,
		Source:          BalanceChangedSourceDeposit,
	})
	return nil
}

func (d *Account) WithDraw(amount float64) error {
	if d == nil {
		return ErrAccounteNotExists
	}
	if d.state.IsDeleted {
		return ErrAccountHasDeleted
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if amount > d.state.Balance {
		return ErrInsufficientBalance
	}
	d.Apply(&AccountBalanceChanged{
		DomainEventBase: domain.NewDomainEventBase(NewUUID()),
		Amount:          -amount,
		Source:          BalanceChangedSourceWithdraw,
	})
	return nil
}

func (d *Account) Remove() error {
	if d == nil || d.state.IsDeleted {
		return nil
	}
	if d.state.Balance != 0 {
		return ErrAccountHasBalance
	}
	d.Apply(&AccountRemoved{
		DomainEventBase: domain.NewDomainEventBase(NewUUID()),
	})
	return nil
}
