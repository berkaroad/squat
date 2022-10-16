package mongoes

import (
	"context"
	"fmt"
	"sync"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/domain/eventsourcing"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ eventsourcing.EventStore = (*MongodbEventStore)(nil)

type MongodbEventStore struct {
	MongoURL string // mongodb connection url, like: mongodb://user:pass@localhost:27012
	DbName   string // database to connect

	initOnce sync.Once
	db       *mongo.Database
}

func (s *MongodbEventStore) GetSnapshot(ctx context.Context, aggregateID string) (eventsourcing.AggregateSnapshotData, error) {
	if s.db == nil {
		panic("field 'db' is null in 'MongodbEventStore'")
	}

	var snapshotData eventsourcing.AggregateSnapshotData
	singleResult := s.db.Collection("aggregatesnapshots").FindOne(ctx, bson.M{
		"aggregateid": aggregateID,
	})
	err := singleResult.Err()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return snapshotData, nil
		}
		return snapshotData, err
	}
	err = singleResult.Decode(&snapshotData)
	if err != nil {
		return snapshotData, err
	}
	return snapshotData, nil
}

func (s *MongodbEventStore) SaveSnapshot(ctx context.Context, data eventsourcing.AggregateSnapshotData) error {
	if s.db == nil {
		panic("field 'db' is null in 'MongodbEventStore'")
	}

	_, err := s.db.Collection("aggregatesnapshots").UpdateOne(ctx,
		bson.M{
			"aggregateid": data.AggregateID,
		},
		bson.M{
			"$set": data,
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return err
	}
	return nil
}

func (s *MongodbEventStore) QueryEventStreamList(ctx context.Context, aggregateID string, startVersion, endVersion int) (eventsourcing.EventStreamDataSlice, error) {
	if s.db == nil {
		panic("field 'db' is null in 'MongodbEventStore'")
	}

	cur, err := s.db.Collection("eventstreams").Find(ctx, bson.M{
		"aggregateid": aggregateID,
		"streamversion": bson.M{
			"$gte": startVersion,
			"$lte": endVersion,
		},
	})
	if err != nil {
		return nil, err
	}
	eventStreamDatas := make(eventsourcing.EventStreamDataSlice, 0)
	err = cur.All(ctx, &eventStreamDatas)
	if err != nil {
		return nil, err
	}
	return eventStreamDatas, nil
}

func (s *MongodbEventStore) AppendEventStream(ctx context.Context, data eventsourcing.EventStreamData) error {
	if s.db == nil {
		panic("field 'db' is null in 'MongodbEventStore'")
	}

	_, err := s.db.Collection("eventstreams").InsertOne(ctx, data)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%w: %v", domain.ErrDuplicateKey, err.Error())
		}
		return err
	}
	return nil
}

func (s *MongodbEventStore) Initialize(ctx context.Context) *MongodbEventStore {
	s.initOnce.Do(func() {
		clientOptions := options.Client().ApplyURI(s.MongoURL)
		client, err := mongo.Connect(ctx, clientOptions)
		if err != nil {
			panic(err)
		}
		err = client.Ping(ctx, nil)
		if err != nil {
			panic(err)
		}
		s.db = client.Database(s.DbName)
	})

	return s
}
