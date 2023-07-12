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

func NewDomainEventBase(eventID string) DomainEventBase {
	return DomainEventBase{
		E_ID: eventID,
		E_Ts: time.Now().Unix(),
	}
}

var _ DomainEvent = DomainEventBase{}

type DomainEventBase struct {
	E_ID string
	E_Ts int64
}

func (e DomainEventBase) TypeName() string {
	panic("method 'TypeName()' not impletement")
}
func (e DomainEventBase) EventID() string {
	return e.E_ID
}
func (e DomainEventBase) OccurTime() time.Time {
	return time.Unix(e.E_Ts, 0)
}

type EventStream struct {
	AggregateID   string
	StreamVersion int
	Events        []DomainEvent
}

type EventStreamSlice []EventStream

func (l EventStreamSlice) Len() int { return len(l) }
func (l EventStreamSlice) Less(i, j int) bool {
	return l[i].AggregateID < l[j].AggregateID || (l[i].AggregateID == l[j].AggregateID && l[i].StreamVersion < l[j].StreamVersion)
}
func (l EventStreamSlice) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
