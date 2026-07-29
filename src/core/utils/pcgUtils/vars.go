package pcgUtils

import (
	"sync"
	"sync/atomic"
)

// global is a parallel pcg for the package functions.
var global PT

// independent incs for each state
var parInc = [...]uint64{
	0x0105c7f8e6e4c8e1,
	0xdd8a45d4a7d3e08e,
	0x8687c0717abf0fce,
	0xfdd14f7a53ba7c6e,
	0xd73bd47d3c1f77f4,
	0xb73f1ab0cfeaf544,
	0x97a106a20fb5466c,
	0xe07d6876e401a906,
}

// a poor man's thread id. use the fact that sync.Pool has some affinity to return
// a counter that should stay the same between calls. it's up to you to turn this
// counter into something useful.

var (
	tidCounter uint64
	tidPool    = sync.Pool{New: func() interface{} { return atomic.AddUint64(&tidCounter, 1) }}
)
