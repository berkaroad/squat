package snapshotstore

import "context"

type SnapshotStore interface {
	GetSnapshot(ctx context.Context, aggregateID string) (AggregateSnapshotData, error)
	SaveSnapshot(ctx context.Context, data AggregateSnapshotData) error
}

type AggregateSnapshotData struct {
	AggregateID       string `json:"aggregate_id" bson:"aggregate_id"`
	AggregateTypeName string `json:"aggregate_type_name" bson:"aggregate_type_name"`
	SnapshotVersion   int    `json:"snapshot_version" bson:"snapshot_version"`
	SnapshotType      string `json:"snapshot_type" bson:"snapshot_type"`
	Body              string `json:"body" bson:"body"`
}
