package eventsourcing

import (
	"fmt"

	"github.com/berkaroad/squat/serialization"
	"github.com/berkaroad/squat/store/snapshotstore"
)

func ToAggregateSnapshot(serializer serialization.Serializer, asd snapshotstore.AggregateSnapshotData) (AggregateSnapshot, error) {
	snapshotObj, err := serialization.Deserialize(serializer, asd.SnapshotType, []byte(asd.Body))
	if err != nil {
		return nil, err
	}
	snapshot, ok := snapshotObj.(AggregateSnapshot)
	if !ok {
		return nil, fmt.Errorf("couldn't cast '%#v' to 'AggregateSnapshot'", snapshotObj)
	}
	return snapshot, nil
}

func ToAggregateSnapshotData(serializer serialization.Serializer, as AggregateSnapshot) (snapshotstore.AggregateSnapshotData, error) {
	asd := snapshotstore.AggregateSnapshotData{
		AggregateID:       as.AggregateID(),
		AggregateTypeName: as.AggregateTypeName(),
		SnapshotVersion:   as.SnapshotVersion(),
		SnapshotType:      as.TypeName(),
	}
	body, err := serialization.Serialize(serializer, as)
	if err != nil {
		return asd, err
	}
	asd.Body = string(body)
	return asd, nil
}
