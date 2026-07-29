package pcg

import (
	"slices"
	"testing"

	"github.com/ALiwoto/codex-dedup/src/core/utils/pcg"
)

func TestPCG(t *testing.T) {
	t.Run("Matches", func(t *testing.T) {
		rng := pcg.New(2345)
		out := make([]uint32, 10)
		for i := range out {
			out[i] = rng.Uint32()
		}

		ok := slices.Equal(out, []uint32{
			0x4fb93cfb,
			0x7f1f4c1e,
			0x9d253788,
			0x424b17a2,
			0x41f308c7,
			0x847fd9fc,
			0x4aa51433,
			0x9f72ee73,
			0x57cb76b4,
			0x8ba782bc,
		})
		if !ok {
			t.Error("wrong out")
		}
	})
}

var (
	blackholeUint32  uint32
	blackholeUint64  uint64
	blackholeFloat32 float32
	blackholeFloat64 float64
)

func BenchmarkPCG(b *testing.B) {
	b.Run("Uint32", func(b *testing.B) {
		rng := pcg.New(2345)
		for i := 0; i < b.N; i++ {
			blackholeUint32 += rng.Uint32()
		}
	})

	b.Run("Uint32n", func(b *testing.B) {
		b.Run("Large", func(b *testing.B) {
			rng := pcg.New(2345)
			for i := 0; i < b.N; i++ {
				blackholeUint32 += rng.Uint32n(1<<31 + 1)
			}
		})

		b.Run("Small", func(b *testing.B) {
			rng := pcg.New(2345)
			for i := 0; i < b.N; i++ {
				blackholeUint32 += rng.Uint32n(1000)
			}
		})
	})

	b.Run("Uint64", func(b *testing.B) {
		rng := pcg.New(2345)
		for i := 0; i < b.N; i++ {
			blackholeUint64 += rng.Uint64()
		}
	})

	b.Run("Float64", func(b *testing.B) {
		rng := pcg.New(2345)
		for i := 0; i < b.N; i++ {
			blackholeFloat64 += rng.Float64()
		}
	})

	b.Run("Float32", func(b *testing.B) {
		rng := pcg.New(2345)
		for i := 0; i < b.N; i++ {
			blackholeFloat32 += rng.Float32()
		}
	})
}
