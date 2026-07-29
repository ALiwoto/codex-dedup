package pcgUtils

// PT is a thread safe pcg generator. The output is non-deterministic, even
// if all of the calls are single threaded. The zero value is valid.
type PT struct {
	state [8]struct {
		v uint64
		_ [120]byte // pad to two cache lines
	}
}

// T is a pcg generator. The zero value is valid.
type T struct{ state uint64 }
