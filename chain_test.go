package deltachain

import "testing"

func TestOneKeyframe(t *testing.T) {
	c := &Chain{}
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "k", Seq: 1, Revision: 1, Keyframe: true})
	got := c.Ready("main", 1, 1)
	if len(got) != 1 || got[0] != "k" {
		t.Fatalf("got %v", got)
	}
}
