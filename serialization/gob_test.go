package serialization

import "testing"

func TestGob(t *testing.T) {
	deserialize(t, &GobSerializer{})
}
