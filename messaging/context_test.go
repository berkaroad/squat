package messaging

import (
	"context"
	"testing"
)

func TestFromContext(t *testing.T) {
	t.Run("message metada should not exists", func(t *testing.T) {
		ctx := context.TODO()
		metadata := FromContext(ctx)
		if metadata != nil {
			t.Error("message metada should not exists")
		}
	})

	t.Run("message metada should exists", func(t *testing.T) {
		ctx := NewContext(context.TODO(), &MessageMetadata{MessageID: "001"})
		metadata := FromContext(ctx)
		if metadata == nil {
			t.Error("message metada should exists")
			return
		}
		if metadata.MessageID != "001" {
			t.Error("message metada should equal with last one")
		}
	})
}
