package crdt

import "testing"

func TestType(t *testing.T) {
	switch val := (&Entity{
		ServiceName: "presence",
		EntityFunc: func(id EntityId) interface{} {
			return nil
		},
		DefaultFunc: func(_ *Context) CRDT {
			return NewFlag()
		},
	}).DefaultFunc(nil).(type) {
	case CRDT:
		if val.State().GetFlag() == nil {
			t.Fatal()
		}
	default:
		t.Fatal()
	}

}
