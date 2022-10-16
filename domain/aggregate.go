package domain

type Aggregate interface {
	AggregateID() string
	AggregateVersion() int
}

var _ Aggregate = (*NullAggregate)(nil)

type NullAggregate struct{}

func (a *NullAggregate) AggregateID() string   { return "" }
func (a *NullAggregate) AggregateVersion() int { return 0 }
