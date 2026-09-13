package plan

import "testing"

func TestFromDisplayScale(t *testing.T) {
	p, err := FromDisplay(
		[]byte(`{"outputs":[{"name":"Virtual-1","scale":2}]}`),
		[]byte(`{"outputs":[{"name":"Virtual-1","width":1920,"height":1080,"refresh_hz":60,"scale":1}]}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Ops) != 1 || p.Ops[0].Scale != 2 || p.Ops[0].Output != "Virtual-1" {
		t.Fatalf("%+v", p.Ops)
	}
}
