package pcgTests

import (
	"testing"

	"github.com/ALiwoto/codex-dedup/src/core/utils/pcg"
)

func BenchmarkTID(b *testing.B) {
	b.Run("Single", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			pcg.GetTid()
		}
	})

	b.Run("Parallel", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				pcg.GetTid()
			}
		})
	})

}
