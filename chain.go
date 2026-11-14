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
type Chain struct {
	frames []Frame
	Work   int
}

func (c *Chain) Add(frame Frame) { c.frames = append(c.frames, frame) }

func (c *Chain) Reset(stream string, epoch int) { _ = stream; _ = epoch }

// Ready returns decodable frame IDs up to the requested sequence.
func (c *Chain) Ready(stream string, epoch, upto int) []string {
	frames := make([]Frame, 0)
	for _, frame := range c.frames {
		c.Work++
		if frame.Epoch <= epoch && frame.Seq <= upto {
			frames = append(frames, frame)
		}
	}
	sort.Slice(frames, func(i, j int) bool { return frames[i].Seq < frames[j].Seq })
	ids := make([]string, 0, len(frames))
	for _, frame := range frames {
		ids = append(ids, frame.ID)
	}
	return ids
}
