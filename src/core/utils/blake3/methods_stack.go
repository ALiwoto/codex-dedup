package blake3

import (
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeAlg"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeValues"
)

func (s *cvstack) pushN(level uint8, vectors *chainVector, count int, flags uint32, key *[8]uint32) {
	for index := range count {
		s.pushL(level, vectors, index)
		for s.bufn == 8 {
			s.flush(flags, key)
		}
	}
}

func (s *cvstack) pushL(level uint8, vectors *chainVector, column int) {
	bit := uint64(1) << (level & 63)
	if s.occ&bit == 0 {
		readChain(vectors, column, &s.stack[level&63])
		s.occ ^= bit
		return
	}

	s.lvls[s.bufn&7] = level
	writeChain(&s.stack[level&63], &s.buf[0], s.bufn)
	copyChain(vectors, column, &s.buf[1], s.bufn)
	s.bufn++
	s.occ ^= bit
}

func (s *cvstack) flush(flags uint32, key *[8]uint32) {
	var output chainVector
	blakeAlg.HashP(
		&s.buf[0],
		&s.buf[1],
		flags|blakeValues.Flag_Parent,
		key,
		&output,
		s.bufn,
	)

	buffered, levels := s.bufn, s.lvls
	s.bufn, s.lvls = 0, [8]uint8{}

	for index := range buffered {
		s.pushL(levels[index]+1, &output, index)
	}
}
