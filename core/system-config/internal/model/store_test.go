package model

import "testing"

func TestSetGetObserved(t *testing.T) {
	s := New()
	if err := s.SetDesired("display", []byte(`{"outputs":[]}`)); err != nil {
		t.Fatal(err)
	}
	if err := s.PutObserved("display", []byte(`{"outputs":[]}`)); err != nil {
		t.Fatal(err)
	}
	d, o, st := s.Get("display")
	if string(d) != `{"outputs":[]}` || string(o) != `{"outputs":[]}` {
		t.Fatalf("got %s / %s", d, o)
	}
	if !st.Present || !st.Configured || st.Changed {
		t.Fatalf("status %+v", st)
	}
}

func TestBluetoothEmptyAdapterNotPresent(t *testing.T) {
	s := New()
	if err := s.PutObserved("bluetooth", []byte(`{"powered":false}`)); err != nil {
		t.Fatal(err)
	}
	_, _, st := s.Get("bluetooth")
	if st.Present {
		t.Fatalf("empty adapter should be present=false: %+v", st)
	}
	if err := s.PutObserved("bluetooth", []byte(`{"powered":true,"adapter":"AA:BB:CC:DD:EE:FF"}`)); err != nil {
		t.Fatal(err)
	}
	_, _, st = s.Get("bluetooth")
	if !st.Present {
		t.Fatal("adapter should be present")
	}
}

func TestChangedWhenDesiredDiffers(t *testing.T) {
	s := New()
	_ = s.PutObserved("display", []byte(`{"outputs":[{"name":"a","scale":1}]}`))
	_ = s.SetDesired("display", []byte(`{"outputs":[{"name":"a","scale":2}]}`))
	_, _, st := s.Get("display")
	if !st.Changed {
		t.Fatal("expected changed")
	}
}
