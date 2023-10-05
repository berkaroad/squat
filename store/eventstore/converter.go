package eventstore

import (
	"fmt"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/serialization"
)

func ToEventStream(serializer serialization.Serializer, esd EventStreamData) (domain.EventStream, error) {
	es := domain.EventStream{
		AggregateID:       esd.AggregateID,
		AggregateTypeName: esd.AggregateTypeName,
		StreamVersion:     esd.StreamVersion,
		Events:            make([]domain.DomainEvent, len(esd.Events)),
		CommandID:         esd.CommandID,
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
		AggregateID:       es.AggregateID,
		AggregateTypeName: es.AggregateTypeName,
		StreamVersion:     es.StreamVersion,
		Events:            make([]DomainEventData, len(es.Events)),
		CommandID:         es.CommandID,
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

func ToEventStreamSlice(serializer serialization.Serializer, esds EventStreamDataSlice) (domain.EventStreamSlice, error) {
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

func ToEventStreamDataSlice(serializer serialization.Serializer, ess domain.EventStreamSlice) (EventStreamDataSlice, error) {
	if len(ess) == 0 {
		return nil, nil
	}
	esds := make(EventStreamDataSlice, len(ess))
	for i, es := range ess {
		esd, err := ToEventStreamData(serializer, es)
		if err != nil {
			return nil, err
		}
		esds[i] = esd
	}
	return esds, nil
}
