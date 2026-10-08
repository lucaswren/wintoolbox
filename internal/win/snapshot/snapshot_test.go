package snapshot

import (
	"errors"
	"testing"
)

type fakeWriter struct {
	values map[string]uint32
	fail   string
}

func (w *fakeWriter) SetDWordValue(key string, value uint32) error {
	if key == w.fail {
		return errors.New("disk failure")
	}
	w.values[key] = value
	return nil
}
func TestIncompleteSnapshotIsNeverPublished(t *testing.T) {
	w := &fakeWriter{values: map[string]uint32{"Saved": 1}, fail: "Policy"}
	if err := Commit(w, map[string]uint32{"Policy": 1, "Service": 4}); err == nil {
		t.Fatal("write failure ignored")
	}
	if w.values["Saved"] != 0 {
		t.Fatal("incomplete snapshot published")
	}
}
func TestCompleteSnapshotPublished(t *testing.T) {
	w := &fakeWriter{values: map[string]uint32{}}
	if err := Commit(w, map[string]uint32{"Policy": 1, "Service": 4}); err != nil {
		t.Fatal(err)
	}
	if w.values["Saved"] != 1 || w.values["Service"] != 4 {
		t.Fatal(w.values)
	}
}
