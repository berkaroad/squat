package serialization

import "testing"

func TestJson(t *testing.T) {
	deserialize(t, &JsonSerializer{})
	deserializeFromText(t, &JsonSerializer{})
}
