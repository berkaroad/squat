package eventsourcing

import "context"

type AggregateSnapshoter[T EventSourcedAggregate] interface {
	TaskSnapshot(ctx context.Context, aggregateID string) error
	GetSnapshot(ctx context.Context, aggregateID string, endVersion int) (T, error)
}
