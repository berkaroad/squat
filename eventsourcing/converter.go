package eventsourcing

import (
	"bytes"
	"fmt"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/serialization"
	"github.com/berkaroad/squat/store/eventstore"
	"github.com/berkaroad/squat/store/snapshotstore"
)

func ToEventStream(serializer serialization.TextSerializer, esd eventstore.EventStreamData) (domain.EventStream, error) {
	es := domain.EventStream{
		AggregateID:   esd.AggregateID,
		AggregateType: esd.AggregateType,
		StreamVersion: esd.StreamVersion,
		Events:        make([]domain.DomainEvent, len(esd.Events)),
		CommandID:     esd.CommandID,
		CommandType:   esd.CommandType,
		Extensions:    esd.Extensions,
	}
	for i, eventData := range esd.Events {
		eventObj, err := serialization.Deserialize(serializer, eventData.EventType, bytes.NewReader([]byte(eventData.Body)))
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

func ToEventStreamData(serializer serialization.TextSerializer, es domain.EventStream) (eventstore.EventStreamData, error) {
	esd := eventstore.EventStreamData{
		AggregateID:   es.AggregateID,
		AggregateType: es.AggregateType,
		StreamVersion: es.StreamVersion,
		Events:        make([]eventstore.DomainEventData, len(es.Events)),
		CommandID:     es.CommandID,
		CommandType:   es.CommandType,
		Extensions:    es.Extensions,
	}
	for i, event := range es.Events {
		body, err := serialization.SerializeToText(serializer, event)
		if err != nil {
			return esd, err
		}
		esd.Events[i] = eventstore.DomainEventData{
			EventID:   event.EventID(),
			EventType: event.TypeName(),
			OccurTime: event.OccurTime().Unix(),
			Body:      body,
		}
	}
	return esd, nil
}

func ToEventStreamSlice(serializer serialization.TextSerializer, esds eventstore.EventStreamDataSlice) (domain.EventStreamSlice, error) {
	if len(esds) == 0 {
		return nil, nil
	}
	ess := make(domain.EventStreamSlice, len(esds))
	for i, esd := range esds {
		es, err := ToEventStream(serializer, esd)
		if err != nil {
			return nil, err
		}
		ess[i] = es
	}
	return ess, nil
}

func ToEventStreamDataSlice(serializer serialization.TextSerializer, ess domain.EventStreamSlice) (eventstore.EventStreamDataSlice, error) {
	if len(ess) == 0 {
		return nil, nil
	}
	esds := make(eventstore.EventStreamDataSlice, len(ess))
	for i, es := range ess {
		esd, err := ToEventStreamData(serializer, es)
		if err != nil {
			return nil, err
		}
		esds[i] = esd
	}
	return esds, nil
}

func ToAggregateSnapshot(serializer serialization.TextSerializer, asd snapshotstore.AggregateSnapshotData) (AggregateSnapshot, error) {
	snapshotObj, err := serialization.Deserialize(serializer, asd.SnapshotType, bytes.NewReader([]byte(asd.Body)))
	if err != nil {
		return nil, err
	}
	snapshot, ok := snapshotObj.(AggregateSnapshot)
	if !ok {
		return nil, fmt.Errorf("couldn't cast '%#v' to 'AggregateSnapshot'", snapshotObj)
	}
	return snapshot, nil
}

func ToAggregateSnapshotData(serializer serialization.TextSerializer, as AggregateSnapshot) (snapshotstore.AggregateSnapshotData, error) {
	asd := snapshotstore.AggregateSnapshotData{
		AggregateID:     as.AggregateID(),
		AggregateType:   as.AggregateTypeName(),
		SnapshotVersion: as.SnapshotVersion(),
		SnapshotType:    as.TypeName(),
	}
	body, err := serialization.SerializeToText(serializer, as)
	if err != nil {
		return asd, err
	}
	asd.Body = body
	return asd, nil
}
