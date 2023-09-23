package domain

import "testing"

func TestNullAggregate(t *testing.T) {
	aggr := NullAggregate{}
	if aggr.AggregateID() != "" {
		t.Error("NullAggregate.AggregateID should empty")
	}
	if aggr.AggregateTypeName() != "" {
		t.Error("NullAggregate.AggregateTypeName should empty")
	}
	if aggr.AggregateVersion() != 0 {
		t.Error("NullAggregate.AggregateVersion should zero")
	}
}
