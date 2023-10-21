package domain

type Aggregate interface {
	AggregateID() string
	AggregateTypeName() string
	AggregateVersion() int
}
