# Spectator delta frame chain

Each stream has independent epochs. A reset advances one stream's epoch and
discards older frames from that stream. Frame IDs are revisioned within an
epoch: older or duplicate revisions cannot replace a newer frame. A keyframe
needs no base; a delta frame is decodable only when its named base frame is
already decodable in the same stream and epoch with a lower sequence number.
`Ready` returns frame IDs up to its sequence limit, in sequence order, omitting
broken chains and unrelated streams. Ties use ID order.

`Work` counts frame candidates examined during reads. A recent read must not
rescan archived epochs or other streams just to build one short chain.
