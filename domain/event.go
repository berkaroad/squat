package domain

import (
	"time"

	"github.com/berkaroad/squat/serialization"
)

type DomainEvent interface {
	serialization.Serializable
	EventID() string
	OccurTime() time.Time
}

// Base of event, should override method 'TypeName()'
func NewDomainEventBase(eventID string) DomainEventBase {
	if eventID == "" {
		panic(ErrEmptyEventID)
	}
	return DomainEventBase{
		E_ID: eventID,
		E_Ts: time.Now().Unix(),
	}
}

type DomainEventBase struct {
	E_ID string `json:"e_id"`
	E_Ts int64  `json:"e_ts"`
}

func (e DomainEventBase) EventID() string {
	return e.E_ID
}
func (e DomainEventBase) OccurTime() time.Time {
	return time.Unix(e.E_Ts, 0)
}

type EventStream struct {
	AggregateID   string
	AggregateType string
	StreamVersion int
	Events        []DomainEvent
	CommandID     string
	CommandType   string
	Extensions    map[string]string
}

func (es *EventStream) Reset() {
	es.AggregateID = ""
	es.AggregateType = ""
	es.StreamVersion = 0
	es.Events = nil
	es.CommandID = ""
	es.CommandType = ""
	es.Extensions = nil
}

type EventStreamSlice []EventStream

func (l EventStreamSlice) Len() int { return len(l) }
func (l EventStreamSlice) Less(i, j int) bool {
	return l[i].AggregateID < l[j].AggregateID || (l[i].AggregateID == l[j].AggregateID && l[i].StreamVersion < l[j].StreamVersion)
}
func (l EventStreamSlice) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
