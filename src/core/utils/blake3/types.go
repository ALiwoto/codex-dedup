package blake3

// Hasher implements hash.Hash for BLAKE3.
type Hasher struct {
	size int
	h    hasher
}

type hasher struct {
	len    uint64
	chunks uint64
	flags  uint32
	key    [8]uint32
	stack  cvstack
	buf    [8192]byte
}

type chainVector = [64]uint32

type cvstack struct {
	occ   uint64
	lvls  [8]uint8
	bufn  int
	buf   [2]chainVector
	stack [64][8]uint32
}

// Digest captures a Hasher snapshot and exposes BLAKE3's extendable output.
type Digest struct {
	counter uint64
	chain   [8]uint32
	block   [16]uint32
	blen    uint32
	flags   uint32
	buf     [16]uint32
	bufn    int
}
