package eventstore

import (
	"context"
)

type EventStore interface {
	QueryEventStreamList(ctx context.Context, aggregateID string, startVersion, endVersion int) (EventStreamDataSlice, error)
	AppendEventStream(ctx context.Context, datas EventStreamDataSlice) error
}

type DomainEventData struct {
	EventID   string `json:"event_id" bson:"event_id"`
	EventType string `json:"event_type" bson:"event_type"`
	OccurTime int64  `json:"occur_time" bson:"occur_time"`
	Body      string `json:"body" bson:"body"`
}

type EventStreamData struct {
	AggregateID   string            `json:"aggregate_id" bson:"aggregate_id"`
	AggregateType string            `json:"aggregate_type" bson:"aggregate_type"`
	StreamVersion int               `json:"stream_version" bson:"stream_version"`
	Events        []DomainEventData `json:"events" bson:"events"`
	CommandID     string            `json:"command_id" bson:"command_id"`
	CommandType   string            `json:"command_type" bson:"command_type"`
	Extensions    map[string]string `json:"extensions" bson:"extensions"`
}

type EventStreamDataSlice []EventStreamData

func (l EventStreamDataSlice) Len() int { return len(l) }
func (l EventStreamDataSlice) Less(i, j int) bool {
	return l[i].AggregateID < l[j].AggregateID || (l[i].AggregateID == l[j].AggregateID && l[i].StreamVersion < l[j].StreamVersion)
}
func (l EventStreamDataSlice) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
