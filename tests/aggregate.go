package tests

import (
	"errors"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/eventsourcing"
)

const (
	AggregateTypeName string = "Account"
	SnapshotTypeName  string = AggregateTypeName + "Snapshot"
)

var _ eventsourcing.EventSourcedAggregate = (*Account)(nil)
var _ eventsourcing.AggregateSnapshot = (*AccountSnapshot)(nil)

type Account struct {
	state AccountState
	eventsourcing.EventSourcedAggregateBase
}

func (a *Account) AggregateID() string {
	return a.state.ID
}

func (a *Account) AggregateTypeName() string {
	return AggregateTypeName
}

func (a *Account) Apply(e domain.DomainEvent) {
	a.EventSourcedAggregateBase.Apply(e, a.mutate)
}

func (a *Account) Snapshot() eventsourcing.AggregateSnapshot {
	return &AccountSnapshot{
		AccountState: a.state,
		Version:      a.AggregateVersion(),
	}
}

func (a *Account) Restore(snapshot eventsourcing.AggregateSnapshot, eventStreams domain.EventStreamSlice) error {
	return a.EventSourcedAggregateBase.Restore(snapshot, a.restoreSnapshot, eventStreams, a.mutate)
}

func (a *Account) mutate(e domain.DomainEvent) {
	switch event := e.(type) {
	case *AccountCreated:
		a.state = AccountState{
			ID:      event.ID,
			Name:    event.Name,
			Balance: event.Balance,
		}
	case *AccountBalanceChanged:
		a.state.Balance += event.Amount
	}
}

func (a *Account) restoreSnapshot(snapshot eventsourcing.AggregateSnapshot) error {
	if snapshot == nil {
		return nil
	}

	if state, ok := snapshot.(*AccountSnapshot); !ok {
		return errors.New("argument 'snapshot' is invalid: convert to '*AccountSnapshot' fail")
	} else {
		a.state = state.AccountState
	}
	return nil
}

type AccountState struct {
	ID      string
	Name    string
	Balance float64

	IsDeleted bool
}

type AccountSnapshot struct {
	AccountState
	Version int
}

func (s *AccountSnapshot) AggregateID() string       { return s.ID }
func (s *AccountSnapshot) AggregateTypeName() string { return AggregateTypeName }
func (s *AccountSnapshot) SnapshotVersion() int      { return s.Version }
func (s *AccountSnapshot) TypeName() string          { return SnapshotTypeName }
