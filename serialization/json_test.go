package serialization

import "testing"

func TestJson(t *testing.T) {
	deserialize(t, &JSONSerializer{})
	deserializeFromText(t, &JSONSerializer{})
}
