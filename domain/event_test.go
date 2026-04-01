package domain

import (
	"math/rand"
	"sort"
	"strconv"
	"testing"
)

func TestNewDomainEventBase(t *testing.T) {
	events := make([]DomainEventBase, 0)
	for i := 0; i < 10; i++ {
		events = append(events, NewDomainEventBase(strconv.Itoa(rand.Int())))
	}

	eIDMapper := make(map[string]struct{})
	for _, event := range events {
		if event.EID == "" {
			t.Errorf("NewDomainEventBase() fail: E_ID is empty")
		}
		if event.ETS <= 0 {
			t.Errorf("NewDomainEventBase() fail: E_Ts is less that or equal to zero")
		}
		if _, ok := eIDMapper[event.EID]; ok {
			t.Errorf("NewDomainEventBase() fail: E_ID is duplicated")
		}
		eIDMapper[event.EID] = struct{}{}
	}
}

func TestDomainEventBase_EventID(t *testing.T) {
	events := make([]DomainEventBase, 0)
	for i := 0; i < 10; i++ {
		events = append(events, NewDomainEventBase(strconv.Itoa(rand.Int())))
	}

	for _, event := range events {
		if event.EID != event.EventID() {
			t.Errorf("NewDomainEventBase.EventID() not equal with E_ID")
		}
	}
}

func TestDomainEventBase_OccurTime(t *testing.T) {
	events := make([]DomainEventBase, 0)
	for i := 0; i < 10; i++ {
		events = append(events, NewDomainEventBase(strconv.Itoa(rand.Int())))
	}

	for _, event := range events {
		if event.ETS != event.OccurTime().Unix() {
			t.Errorf("NewDomainEventBase.OccurTime() not equal with E_Ts")
		}
	}
}

func TestSortEventStreamSlice(t *testing.T) {
	eventstreams := make(EventStreamSlice, 0)
	for i := 10; i >= 1; i-- {
		eventstreams = append(eventstreams, EventStream{
			AggregateID:   "001",
			AggregateType: "fruit",
			StreamVersion: i,
			Events:        []DomainEvent{&sampleDomainEvent{DomainEventBase: NewDomainEventBase(strconv.Itoa(rand.Int()))}},
			CommandID:     "command-001",
			CommandType:   "commandType1",
		})
	}

	sort.Sort(eventstreams)
	for i := 0; i < 10; i++ {
		if eventstreams[i].StreamVersion != i+1 {
			t.Errorf("eventstreams should sort by 'StreamVersion")
		}
	}
}

var _ DomainEvent = (*sampleDomainEvent)(nil)

type sampleDomainEvent struct {
	DomainEventBase
}

// TypeName implements [DomainEvent].
func (s *sampleDomainEvent) TypeName() string {
	return "sampleDomainEvent"
}
