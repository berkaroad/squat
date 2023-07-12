package domain

import (
	"math/rand"
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
		if event.E_ID == "" {
			t.Errorf("NewDomainEventBase() fail: E_ID is empty")
		}
		if event.E_Ts <= 0 {
			t.Errorf("NewDomainEventBase() fail: E_Ts is less that or equal to zero")
		}
		if _, ok := eIDMapper[event.E_ID]; ok {
			t.Errorf("NewDomainEventBase() fail: E_ID is duplicated")
		}
		eIDMapper[event.E_ID] = struct{}{}
	}
}

func TestDomainEventBase_EventID(t *testing.T) {
	events := make([]DomainEventBase, 0)
	for i := 0; i < 10; i++ {
		events = append(events, NewDomainEventBase(strconv.Itoa(rand.Int())))
	}

	for _, event := range events {
		if event.E_ID != event.EventID() {
			t.Errorf("NewDomainEventBase.EventID() not equal with E_ID")
		}
	}
}

func TestDomainEventBase_TypeName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("NewDomainEventBase.TypeName() should panic")
		}
	}()

	event := NewDomainEventBase(strconv.Itoa(rand.Int()))
	event.TypeName()
}

func TestDomainEventBase_OccurTime(t *testing.T) {
	events := make([]DomainEventBase, 0)
	for i := 0; i < 10; i++ {
		events = append(events, NewDomainEventBase(strconv.Itoa(rand.Int())))
	}

	for _, event := range events {
		if event.E_Ts != event.OccurTime().Unix() {
			t.Errorf("NewDomainEventBase.OccurTime() not equal with E_Ts")
		}
	}
}
