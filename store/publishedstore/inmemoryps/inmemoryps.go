package inmemoryps

import (
	"context"
	"sync"
	"time"

	"github.com/berkaroad/squat/store/publishedstore"
)

var instance *InMemoryPublishedStore = &InMemoryPublishedStore{SimulateTimeout: time.Millisecond * 20}

func Default() *InMemoryPublishedStore {
	return instance
}

var _ publishedstore.PublishedStore = (*InMemoryPublishedStore)(nil)

type InMemoryPublishedStore struct {
	SimulateTimeout time.Duration

	store sync.Map
}

func (s *InMemoryPublishedStore) GetPublishedVersion(ctx context.Context, aggregateID string) (int, error) {
	time.Sleep(s.SimulateTimeout)
	if val, ok := s.store.Load(aggregateID); ok {
		exists := val.(*publishedstore.PublishedEventStreamRef)
		return exists.PublishedVersion, nil
	} else {
		return 0, nil
	}
}

func (s *InMemoryPublishedStore) SavePublished(ctx context.Context, datas []publishedstore.PublishedEventStreamRef) error {
	time.Sleep(s.SimulateTimeout)
	for _, data := range datas {
		actual, loaded := s.store.LoadOrStore(data.AggregateID, &data)
		if loaded {
			exists := actual.(*publishedstore.PublishedEventStreamRef)
			exists.PublishedVersion = data.PublishedVersion
		}
	}
	return nil
}
