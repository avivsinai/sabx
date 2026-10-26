package sabapi

import (
	"encoding/json"
	"testing"
)

func TestHistorySlotAcceptsNumericAndStringFields(t *testing.T) {
	cases := map[string]string{
		"number": `{"name":"a","completed":1793039228,"bytes":2147483648}`,
		"string": `{"name":"a","completed":"1793039228","bytes":"2147483648"}`,
	}
	for name, in := range cases {
		var slot HistorySlot
		if err := json.Unmarshal([]byte(in), &slot); err != nil {
			t.Fatalf("%s: unmarshal: %v", name, err)
		}
		if slot.Completed != "1793039228" || slot.Bytes != "2147483648" {
			t.Fatalf("%s: got completed=%q bytes=%q", name, slot.Completed, slot.Bytes)
		}
	}
	var slot HistorySlot
	if err := json.Unmarshal([]byte(`{"completed":null}`), &slot); err != nil || slot.Completed != "" {
		t.Fatalf("null: err=%v completed=%q", err, slot.Completed)
	}
	if err := json.Unmarshal([]byte(`{"completed":{}}`), &slot); err == nil {
		t.Fatal("object: expected error")
	}
}
