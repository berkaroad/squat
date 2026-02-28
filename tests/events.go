package tests

import (
	"github.com/berkaroad/squat/domain"
)

const (
	Event_AccountCreated        string = "AccountCreated"
	Event_AccountBalanceChanged string = "AccountBalanceChanged"
	Event_AccountRemoved        string = "AccountRemoved"
)

var _ domain.DomainEvent = (*AccountCreated)(nil)

type AccountCreated struct {
	domain.DomainEventBase

	ID      string
	Name    string
	Balance float64
}

func (e AccountCreated) TypeName() string {
	return Event_AccountCreated
}

var _ domain.DomainEvent = (*AccountBalanceChanged)(nil)

type AccountBalanceChanged struct {
	domain.DomainEventBase

	Source BalanceChangedSource
	Amount float64
}

func (e AccountBalanceChanged) TypeName() string {
	return Event_AccountBalanceChanged
}

type BalanceChangedSource string

const (
	BalanceChangedSourceDeposit  BalanceChangedSource = "deposit"
	BalanceChangedSourceWithdraw BalanceChangedSource = "withdraw"
	BalanceChangedSourceTransfer BalanceChangedSource = "transfer"
)

var _ domain.DomainEvent = (*AccountRemoved)(nil)

type AccountRemoved struct {
	domain.DomainEventBase
}

func (e AccountRemoved) TypeName() string {
	return Event_AccountRemoved
}
