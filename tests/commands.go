package tests

import (
	"github.com/berkaroad/squat/commanding"
)

const (
	Command_CreateAccount string = "CreateAccount"
	Command_Deposit       string = "Deposit"
	Command_Withdraw      string = "Withdraw"
	Command_RemoveAccount string = "RemoveAccount"
)

var _ commanding.Command = (*CreateAccountCommand)(nil)

type CreateAccountCommand struct {
	commanding.CommandBase

	Name string
}

func (c CreateAccountCommand) TypeName() string {
	return Command_CreateAccount
}

func (c CreateAccountCommand) AggregateTypeName() string {
	return AggregateTypeName
}

var _ commanding.Command = (*DepositCommand)(nil)

type DepositCommand struct {
	commanding.CommandBase

	Amount float64
}

func (c DepositCommand) TypeName() string {
	return Command_Deposit
}

func (c DepositCommand) AggregateTypeName() string {
	return AggregateTypeName
}

var _ commanding.Command = (*WithdrawCommand)(nil)

type WithdrawCommand struct {
	commanding.CommandBase

	Amount float64
}

func (c WithdrawCommand) TypeName() string {
	return Command_Withdraw
}

func (c WithdrawCommand) AggregateTypeName() string {
	return AggregateTypeName
}

var _ commanding.Command = (*RemoveAccountCommand)(nil)

type RemoveAccountCommand struct {
	commanding.CommandBase
}

func (c RemoveAccountCommand) TypeName() string {
	return Command_RemoveAccount
}

func (c RemoveAccountCommand) AggregateTypeName() string {
	return AggregateTypeName
}
