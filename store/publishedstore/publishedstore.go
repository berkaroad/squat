package publishedstore

import (
	"context"
)

type PublishedStore interface {
	GetPublishedVersion(ctx context.Context, aggregateID string) (int, error)
	SavePublished(ctx context.Context, datas []PublishedEventStreamRef) error
}

type PublishedEventStreamRef struct {
	AggregateID       string `json:"aggregate_id" bson:"aggregate_id"`
	AggregateTypeName string `json:"aggregate_type_name" bson:"aggregate_type_name"`
	PublishedVersion  int    `json:"published_version" bson:"published_version"`
}
