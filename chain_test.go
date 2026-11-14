package deltachain

import (
	"slices"
	"strconv"
	"testing"
)

func TestOneKeyframe(t *testing.T) {
	c := &Chain{}
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "k", Seq: 1, Revision: 1, Keyframe: true})
	got := c.Ready("main", 1, 1)
	if len(got) != 1 || got[0] != "k" {
		t.Fatalf("got %v", got)
	}
}

func TestDeltaWithoutBaseIsSkipped(t *testing.T) {
	c := &Chain{}
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "d1", Base: "missing", Seq: 2, Revision: 1})
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "k", Seq: 1, Revision: 1, Keyframe: true})
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "d2", Base: "d1", Seq: 3, Revision: 1})

	got := c.Ready("main", 1, 10)
	want := []string{"k"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v (dangling delta and its dependents must be skipped)", got, want)
	}
}

func TestDecodableChainInOrder(t *testing.T) {
	c := &Chain{}
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "d2", Base: "d1", Seq: 3, Revision: 1})
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "k", Seq: 1, Revision: 1, Keyframe: true})
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "d1", Base: "k", Seq: 2, Revision: 1})

	got := c.Ready("main", 1, 10)
	want := []string{"k", "d1", "d2"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestUptoLimit(t *testing.T) {
	c := &Chain{}
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "k", Seq: 1, Revision: 1, Keyframe: true})
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "d1", Base: "k", Seq: 2, Revision: 1})
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "d2", Base: "d1", Seq: 3, Revision: 1})

	got := c.Ready("main", 1, 2)
	want := []string{"k", "d1"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestNewerRevisionWins(t *testing.T) {
	c := &Chain{}
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "k", Seq: 1, Revision: 1, Keyframe: true})
	// Old revision of d1: points at a nonexistent base and would be unplayable.
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "d1", Base: "ghost", Seq: 2, Revision: 1})
	// Newer revision re-points the same ID at the real keyframe.
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "d1", Base: "k", Seq: 2, Revision: 2})
	// Re-sending the stale revision must not overwrite the newer one.
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "d1", Base: "ghost", Seq: 2, Revision: 1})

	got := c.Ready("main", 1, 10)
	want := []string{"k", "d1"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v (newer revision must win and stay)", got, want)
	}
}

func TestResetInvalidatesOldEpoch(t *testing.T) {
	c := &Chain{}
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "old-k", Seq: 1, Revision: 1, Keyframe: true})
	c.Reset("main", 2)
	c.Add(Frame{Stream: "main", Epoch: 2, ID: "new-k", Seq: 1, Revision: 1, Keyframe: true})

	if got := c.Ready("main", 1, 10); len(got) != 0 {
		t.Fatalf("old epoch after reset: got %v, want []", got)
	}
	got := c.Ready("main", 2, 10)
	if !slices.Equal(got, []string{"new-k"}) {
		t.Fatalf("new epoch: got %v, want [new-k]", got)
	}
}

func TestStreamsDoNotCrossChain(t *testing.T) {
	c := &Chain{}
	c.Add(Frame{Stream: "a", Epoch: 1, ID: "k", Seq: 1, Revision: 1, Keyframe: true})
	// Same ID space in another stream must not serve as a base.
	c.Add(Frame{Stream: "b", Epoch: 1, ID: "k", Seq: 1, Revision: 1, Keyframe: true})
	c.Add(Frame{Stream: "b", Epoch: 1, ID: "d", Base: "k", Seq: 2, Revision: 1})

	gotA := c.Ready("a", 1, 10)
	if !slices.Equal(gotA, []string{"k"}) {
		t.Fatalf("stream a: got %v, want [k]", gotA)
	}
	gotB := c.Ready("b", 1, 10)
	if !slices.Equal(gotB, []string{"k", "d"}) {
		t.Fatalf("stream b: got %v, want [k d]", gotB)
	}

	// A delta whose base only exists in another epoch stays broken.
	c.Add(Frame{Stream: "a", Epoch: 2, ID: "lonely", Base: "k", Seq: 1, Revision: 1})
	if got := c.Ready("a", 2, 10); len(got) != 0 {
		t.Fatalf("cross-epoch base: got %v, want []", got)
	}
}

func TestTieSequenceUsesIDOrder(t *testing.T) {
	c := &Chain{}
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "z", Seq: 1, Revision: 1, Keyframe: true})
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "a", Seq: 1, Revision: 1, Keyframe: true})
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "m", Seq: 1, Revision: 1, Keyframe: true})

	got := c.Ready("main", 1, 10)
	want := []string{"a", "m", "z"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBaseMustHaveLowerSequence(t *testing.T) {
	c := &Chain{}
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "k", Seq: 1, Revision: 1, Keyframe: true})
	// Same-sequence base cannot anchor a delta: processing order is by Seq,
	// and the base is not decodable in an earlier pass.
	c.Add(Frame{Stream: "main", Epoch: 1, ID: "loop", Base: "loop", Seq: 2, Revision: 1})

	got := c.Ready("main", 1, 10)
	if !slices.Equal(got, []string{"k"}) {
		t.Fatalf("got %v, want [k] (self/same-seq reference is not a base)", got)
	}
}

func TestRecentReadsDoNotRescanArchives(t *testing.T) {
	c := &Chain{}

	// 3000 frames parked in old, archived epochs.
	for i := 0; i < 3000; i++ {
		c.Add(Frame{
			Stream:   "main",
			Epoch:    1,
			ID:       "old-" + strconv.Itoa(i),
			Seq:      i + 1,
			Revision: 1,
			Keyframe: i == 0,
		})
	}
	c.Reset("main", 2)

	// A short recent chain in the live epoch.
	c.Add(Frame{Stream: "main", Epoch: 2, ID: "k", Seq: 1, Revision: 1, Keyframe: true})
	c.Add(Frame{Stream: "main", Epoch: 2, ID: "d", Base: "k", Seq: 2, Revision: 1})

	for i := 0; i < 200; i++ {
		got := c.Ready("main", 2, 10)
		if !slices.Equal(got, []string{"k", "d"}) {
			t.Fatalf("got %v, want [k d]", got)
		}
	}

	if c.Work > 6000 {
		t.Fatalf("Work = %d, must not exceed 6000 for 200 recent short-chain reads", c.Work)
	}
}
