package pcg

// Uint32 returns a random uint32.
// Safe for concurrent callers.
func Uint32() uint32 {
	state := global.next()

	xor := uint32(((state >> 18) ^ state) >> 27)
	shift := uint(state>>59) & 31

	return xor>>shift | xor<<(32-shift)
}

// Uint32n returns a uint32 uniformly in [0, n).
// Safe for concurrent callers.
func Uint32n(n uint32) uint32 {
	if n == 0 {
		return 0
	}

	x := global.Uint32()
	m := uint64(x) * uint64(n)
	l := uint32(m)

	if l < n {
		t := -n
		if t >= n {
			t -= n
			if t >= n {
				t = t % n
			}
		}

	again:
		if l < t {
			x = global.Uint32()
			m = uint64(x) * uint64(n)
			l = uint32(m)
			goto again
		}
	}

	return uint32(m >> 32)
}

// Uint64 returns a random uint64.
// Safe for concurrent callers.
func Uint64() uint64 {
	state1 := global.next()
	state2 := global.next()

	xor1 := uint32(((state1 >> 18) ^ state1) >> 27)
	shift1 := uint(state1>>59) & 31

	xor2 := uint32(((state2 >> 18) ^ state2) >> 27)
	shift2 := uint(state2>>59) & 31

	return uint64(xor1>>shift1|xor1<<(32-shift1))<<32 |
		uint64(xor2>>shift2|xor2<<(32-shift2))
}

// Float64 returns a float64 uniformly in [0, 1).
// Safe for concurrent callers.
func Float64() float64 {
again:
	state1 := global.next()
	state2 := global.next()

	xor1 := uint32(((state1 >> 18) ^ state1) >> 27)
	shift1 := uint(state1>>59) & 31

	xor2 := uint32(((state2 >> 18) ^ state2) >> 27)
	shift2 := uint(state2>>59) & 31

	v := uint64(xor1>>shift1|xor1<<(32-shift1)) |
		uint64(xor2>>shift2|xor2<<(32-shift2))

	out := float64(v>>(64-53)) / (1 << 53)
	if out == 1 {
		goto again
	}

	return out
}

// Float32 returns a float32 uniformly in [0, 1).
// Safe for concurrent callers.
func Float32() float32 {
again:
	out := float32(global.Uint32()>>(32-24)) / (1 << 24)
	if out == 1 {
		goto again
	}

	return out
}

// New constructs a parallel pcg with the given state.
func NewParallel(state uint64) PT {
	var pt PT
	pt.state[0].v = state + 0
	pt.state[1].v = state + 1
	pt.state[2].v = state + 2
	pt.state[3].v = state + 3
	pt.state[4].v = state + 4
	pt.state[5].v = state + 5
	pt.state[6].v = state + 6
	pt.state[7].v = state + 7
	return pt
}

// New constructs a pcg with the given state.
func New(state uint64) T { return T{state} }

func GetTid() (v uint64) {
	x := tidPool.Get()
	tidPool.Put(x)
	v, _ = x.(uint64)
	return v
}
