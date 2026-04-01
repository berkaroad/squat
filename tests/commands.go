package tests

import (
	"github.com/berkaroad/squat/commanding"
)

const (
	CommandCreateAccount string = "CreateAccount"
	CommandDeposit       string = "Deposit"
	CommandWithdraw      string = "Withdraw"
	CommandRemoveAccount string = "RemoveAccount"
)

var _ commanding.Command = (*CreateAccountCommand)(nil)

type CreateAccountCommand struct {
	commanding.CommandBase
	AccountID string
	Name      string
}

func (c CreateAccountCommand) TypeName() string {
	return CommandCreateAccount
}

func (c CreateAccountCommand) AggregateID() string {
	return c.AccountID
}

func (c CreateAccountCommand) AggregateTypeName() string {
	return AggregateTypeName
}

var _ commanding.Command = (*DepositCommand)(nil)

type DepositCommand struct {
	commanding.CommandBase
	AccountID string
	Amount    float64
}

func (c DepositCommand) TypeName() string {
	return CommandDeposit
}

func (c DepositCommand) AggregateID() string {
	return c.AccountID
}

func (c DepositCommand) AggregateTypeName() string {
	return AggregateTypeName
}

var _ commanding.Command = (*WithdrawCommand)(nil)

type WithdrawCommand struct {
	commanding.CommandBase
	AccountID string
	Amount    float64
}

func (c WithdrawCommand) TypeName() string {
	return CommandWithdraw
}

func (c WithdrawCommand) AggregateID() string {
	return c.AccountID
}

func (c WithdrawCommand) AggregateTypeName() string {
	return AggregateTypeName
}

var _ commanding.Command = (*RemoveAccountCommand)(nil)

type RemoveAccountCommand struct {
	commanding.CommandBase
	AccountID string
}

func (c RemoveAccountCommand) TypeName() string {
	return CommandRemoveAccount
}

func (c RemoveAccountCommand) AggregateID() string {
	return c.AccountID
}

func (c RemoveAccountCommand) AggregateTypeName() string {
	return AggregateTypeName
}
