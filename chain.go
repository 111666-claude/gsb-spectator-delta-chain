package deltachain

import "sort"

// Frame is a keyframe or a delta that depends on another frame ID.
type Frame struct {
	Stream   string
	Epoch    int
	ID       string
	Base     string
	Seq      int
	Revision int
	Keyframe bool
}

// Chain plans frames the spectator decoder may consume.
//
// Frames are bucketed by stream and epoch so a read only touches the bucket it
// asks for: archived epochs and unrelated streams are never rescanned. Within
// a bucket, frame IDs map to their newest revision.
type Chain struct {
	// buckets maps stream -> epoch -> frame ID -> newest-revision frame.
	buckets map[string]map[int]map[string]Frame
	// Work counts frame candidates examined during reads.
	Work int
}

// Add records a frame. A frame whose ID already exists in the same stream and
// epoch is replaced only when its revision is newer; older or equal revisions
// are ignored so they cannot supersede a newer frame.
func (c *Chain) Add(frame Frame) {
	if c.buckets == nil {
		c.buckets = make(map[string]map[int]map[string]Frame)
	}
	epochs, ok := c.buckets[frame.Stream]
	if !ok {
		epochs = make(map[int]map[string]Frame)
		c.buckets[frame.Stream] = epochs
	}
	bucket, ok := epochs[frame.Epoch]
	if !ok {
		bucket = make(map[string]Frame)
		epochs[frame.Epoch] = bucket
	}
	if current, ok := bucket[frame.ID]; ok && current.Revision >= frame.Revision {
		return
	}
	bucket[frame.ID] = frame
}

// Reset advances one stream's epoch and discards every older frame that
// stream ever published, so frames from before the reset can never be read or
// used as a base again. Other streams are untouched.
func (c *Chain) Reset(stream string, epoch int) {
	epochs, ok := c.buckets[stream]
	if !ok {
		return
	}
	for e := range epochs {
		if e < epoch {
			delete(epochs, e)
		}
	}
}

// Ready returns decodable frame IDs for the given stream and epoch, up to the
// requested sequence, in sequence order (ID order on ties). Keyframes are
// always decodable; a delta is included only when its named base is a
// decodable frame in the same stream and epoch with a lower sequence number.
// Frames from other streams, other epochs, or broken chains are omitted.
func (c *Chain) Ready(stream string, epoch, upto int) []string {
	ids := make([]string, 0)

	epochs, ok := c.buckets[stream]
	if !ok {
		return ids
	}
	bucket, ok := epochs[epoch]
	if !ok {
		return ids
	}

	// Only candidates in the requested bucket are examined; archived epochs
	// and other streams are never scanned.
	candidates := make([]Frame, 0, len(bucket))
	for _, frame := range bucket {
		c.Work++
		if frame.Seq <= upto {
			candidates = append(candidates, frame)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Seq != candidates[j].Seq {
			return candidates[i].Seq < candidates[j].Seq
		}
		return candidates[i].ID < candidates[j].ID
	})

	decodable := make(map[string]bool, len(candidates))
	for _, frame := range candidates {
		switch {
		case frame.Keyframe:
			decodable[frame.ID] = true
		case frame.Base != "" &&
			frame.Base != frame.ID &&
			decodable[frame.Base]:
			// The base lives in this same bucket (same stream and epoch) and,
			// being decodable in an earlier pass, has a lower sequence.
			decodable[frame.ID] = true
		}
		if decodable[frame.ID] {
			ids = append(ids, frame.ID)
		}
	}
	return ids
}
