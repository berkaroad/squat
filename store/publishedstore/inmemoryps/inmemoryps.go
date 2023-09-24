package inmemoryps

import (
	"context"
	"sync"

	"github.com/berkaroad/squat/store/publishedstore"
)

var _ publishedstore.PublishedStore = (*InMemoryPublishedStore)(nil)

type InMemoryPublishedStore struct {
	store sync.Map
}

func (s *InMemoryPublishedStore) GetPublishedVersion(ctx context.Context, aggregateID string) (int, error) {
	if val, ok := s.store.Load(aggregateID); ok {
		exists := val.(*publishedstore.PublishedEventStreamRef)
		return exists.PublishedVersion, nil
	} else {
		return 0, nil
	}
}

func (s *InMemoryPublishedStore) Save(ctx context.Context, data publishedstore.PublishedEventStreamRef) error {
	actual, loaded := s.store.LoadOrStore(data.AggregateID, &data)
	if loaded {
		exists := actual.(*publishedstore.PublishedEventStreamRef)
		exists.PublishedVersion = data.PublishedVersion
	}
	return nil
}
