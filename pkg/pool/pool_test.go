package pool

import "testing"

type testObject struct {
	value  int
	resets int
}

func (object *testObject) Reset() {
	object.value = 0
	object.resets++
}

func TestPool(t *testing.T) {
	created := 0
	pool := New(func() *testObject {
		created++
		return &testObject{}
	})

	object := pool.Get()
	if object == nil || created != 1 {
		t.Fatalf("Get did not create an object: object=%v, created=%d", object, created)
	}
	object.value = 42
	pool.Put(object)
	if object.value != 0 || object.resets != 1 {
		t.Fatalf("Put did not reset the object: %+v", object)
	}

	if next := pool.Get(); next == nil || next.value != 0 {
		t.Fatalf("Get returned an invalid object: %+v", next)
	}
}

func TestNewRejectsNilFactory(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("New accepted a nil factory")
		}
	}()
	New[*testObject](nil)
}
