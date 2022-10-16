package domain

import "context"

type Repository[T Aggregate] interface {
	Get(ctx context.Context, aggregateID string) (T, error)
	Save(ctx context.Context, aggregate T) error
}
