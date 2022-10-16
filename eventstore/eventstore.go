package eventstore

import (
	"context"
)

type EventStore interface {
	GetSnapshot(ctx context.Context, aggregateID string) (AggregateSnapshotData, error)
	SaveSnapshot(ctx context.Context, data AggregateSnapshotData) error
	QueryEventStreamList(ctx context.Context, aggregateID string, startVersion, endVersion int) (EventStreamDataSlice, error)
	AppendEventStream(ctx context.Context, data EventStreamData) error
}

type AggregateSnapshotData struct {
	AggregateID     string `json:"AggregateID"`
	SnapshotVersion int    `json:"SnapshotVersion"`
	SnapshotType    string `json:"SnapshotType"`
	Body            string `json:"Body"`
}

type DomainEventData struct {
	EventID   string `json:"EventID"`
	EventType string `json:"EventType"`
	OccurTime int64  `json:"OccurTime"`
	Body      string `json:"Body"`
}

type EventStreamData struct {
	AggregateID   string            `json:"AggregateID"`
	StreamVersion int               `json:"StreamVersion"`
	Events        []DomainEventData `json:"Events"`
}

type EventStreamDataSlice []EventStreamData

func (l EventStreamDataSlice) Len() int { return len(l) }
func (l EventStreamDataSlice) Less(i, j int) bool {
	return l[i].AggregateID < l[j].AggregateID || (l[i].AggregateID == l[j].AggregateID && l[i].StreamVersion < l[j].StreamVersion)
}
func (l EventStreamDataSlice) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
