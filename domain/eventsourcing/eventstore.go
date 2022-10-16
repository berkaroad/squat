package eventsourcing

import (
	"context"
	"fmt"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/serialization"
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

func ToAggregateSnapshot(serializer serialization.Serializer, asd AggregateSnapshotData) (AggregateSnapshot, error) {
	snapshotObj, err := serialization.Deserialize(serializer, asd.SnapshotType, []byte(asd.Body))
	if err != nil {
		return nil, err
	}
	snapshot, ok := snapshotObj.(AggregateSnapshot)
	if !ok {
		return nil, fmt.Errorf("cann't cast '%#v' to 'eventsourcing.AggregateSnapshot'", snapshotObj)
	}
	return snapshot, nil
}

func ToAggregateSnapshotData(serializer serialization.Serializer, as AggregateSnapshot) (AggregateSnapshotData, error) {
	asd := AggregateSnapshotData{
		AggregateID:     as.AggregateID(),
		SnapshotVersion: as.SnapshotVersion(),
		SnapshotType:    as.TypeName(),
	}
	body, err := serialization.Serialize(serializer, as)
	if err != nil {
		return asd, err
	}
	asd.Body = string(body)
	return asd, nil
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

func ToEventStream(serializer serialization.Serializer, esd EventStreamData) (domain.EventStream, error) {
	es := domain.EventStream{
		AggregateID:   esd.AggregateID,
		StreamVersion: esd.StreamVersion,
		Events:        make([]domain.DomainEvent, len(esd.Events)),
	}
	for i, eventData := range esd.Events {
		eventObj, err := serialization.Deserialize(serializer, eventData.EventType, []byte(eventData.Body))
		if err != nil {
			return es, err
		}
		event, ok := eventObj.(domain.DomainEvent)
		if !ok {
			return es, fmt.Errorf("cann't cast '%#v' to 'domain.DomainEvent'", eventObj)
		}
		es.Events[i] = event
	}
	return es, nil
}

func ToEventStreamData(serializer serialization.Serializer, es domain.EventStream) (EventStreamData, error) {
	esd := EventStreamData{
		AggregateID:   es.AggregateID,
		StreamVersion: es.StreamVersion,
		Events:        make([]DomainEventData, len(es.Events)),
	}
	for i, event := range es.Events {
		body, err := serialization.Serialize(serializer, event)
		if err != nil {
			return esd, err
		}
		esd.Events[i] = DomainEventData{
			EventID:   event.EventID(),
			EventType: event.TypeName(),
			OccurTime: event.OccurTime().Unix(),
			Body:      string(body),
		}
	}
	return esd, nil
}
