package tests

import (
	"context"
	"fmt"

	"github.com/berkaroad/squat/commanding"
	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/logging"
)

var _ commanding.CommandHandlerGroup = (*AccountCommandHandler)(nil)

type AccountCommandHandler struct {
	Repo domain.Repository[*Account]
}

func (ch *AccountCommandHandler) CommandHandlers() map[string]commanding.CommandHandler {
	return map[string]commanding.CommandHandler{
		Command_CreateAccount: {Handle: ch.handleCreateAccount},
		Command_Deposit:       {Handle: ch.handleDeposit},
		Command_Withdraw:      {Handle: ch.handleWithdraw},
		Command_RemoveAccount: {Handle: ch.handleRemoveAccount},
	}
}

func (ch *AccountCommandHandler) handleCreateAccount(ctx context.Context, data commanding.CommandData) error {
	logger := logging.Get(ctx)
	data.SetCustomExtension(ctx, "create", "false")
	cmd := data.Command.(*CreateAccountCommand)

	accountData, err := ch.Repo.Get(ctx, cmd.C_AggregateID)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	newAccountData, err := CreateAccount(accountData, AccountInfo{
		ID:   cmd.C_AggregateID,
		Name: cmd.Name,
	})
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	err = ch.Repo.Save(ctx, newAccountData)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	return nil
}

func (ch *AccountCommandHandler) handleDeposit(ctx context.Context, data commanding.CommandData) error {
	logger := logging.Get(ctx)
	data.SetCustomExtension(ctx, "update", "true")
	cmd := data.Command.(*DepositCommand)

	accountData, err := ch.Repo.Get(ctx, cmd.C_AggregateID)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	if accountData == nil {
		return fmt.Errorf("%w: %s", ErrAccounteNotExists, cmd.C_AggregateID)
	}

	err = accountData.Deposit(cmd.Amount)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	err = ch.Repo.Save(ctx, accountData)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	return nil
}

func (ch *AccountCommandHandler) handleWithdraw(ctx context.Context, data commanding.CommandData) error {
	logger := logging.Get(ctx)
	data.SetCustomExtension(ctx, "update", "true")
	cmd := data.Command.(*WithdrawCommand)

	accountData, err := ch.Repo.Get(ctx, cmd.C_AggregateID)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	if accountData == nil {
		return fmt.Errorf("%w: %s", ErrAccounteNotExists, cmd.C_AggregateID)
	}

	err = accountData.WithDraw(cmd.Amount)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	err = ch.Repo.Save(ctx, accountData)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	return nil
}

func (ch *AccountCommandHandler) handleRemoveAccount(ctx context.Context, data commanding.CommandData) error {
	logger := logging.Get(ctx)
	data.SetCustomExtension(ctx, "delete", "true")
	cmd := data.Command.(*RemoveAccountCommand)

	accountData, err := ch.Repo.Get(ctx, cmd.C_AggregateID)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	if accountData == nil {
		return nil
	}

	err = accountData.Remove()
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	err = ch.Repo.Save(ctx, accountData)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	return nil
}
