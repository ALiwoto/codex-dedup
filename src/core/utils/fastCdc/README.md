# FastCDC chunker

This package implements the project's deterministic `fastcdc-v1` content-defined
chunking algorithm. It treats input as opaque bytes and computes BLAKE3-256 for
every chunk and for the complete stream.

The initial defaults are:

- minimum: 16 KiB
- target average: 64 KiB
- maximum: 256 KiB

The boundary detector follows FastCDC's Gear rolling hash and two-mask
normalization design. The normalization level and Gear table seed are part of
`fastcdc-v1`, not runtime options, so two implementations using the same three
sizes cannot silently choose different boundaries. The Gear table is generated
deterministically with SplitMix64 during package initialization.

The exact `fastcdc-v1` boundary calculation is:

1. Generate 256 Gear values with SplitMix64, beginning with state
   `0x6a09e667f3bcc909` and using the constants in `constants.go`.
2. Let `bits` be `floor(log2(average))`. The mask used below the target is
   `(1 << (bits + 2)) - 1`; the mask used at and above it is
   `(1 << (bits - 2)) - 1`.
3. Skip the first `minimum` bytes. For each following byte, update
   `hash = (hash << 1) + gear[byte]`. Cut immediately after a byte for which
   `hash & mask == 0`, switching masks at `average`.
4. Force a cut at `maximum`. The final chunk ends at EOF and may be smaller
   than `minimum`.

The boundary fixture in `tests/fastCdcTests` prevents accidental changes to
this definition.

`Chunk.Data` aliases a bounded internal buffer and remains valid only until the
next call to `Next` or `Reset`. The buffer is twice the configured maximum chunk
size. The final chunk may be smaller than the configured minimum.

FastCDC was introduced in Wen Xia et al., "FastCDC: a Fast and Efficient
Content-Defined Chunking Approach for Data Deduplication," USENIX ATC 2016.

Run the default and parameter-comparison benchmarks with:

```powershell
go test -run '^$' -bench '^BenchmarkFastCdc' -benchmem ./tests/fastCdcTests
```
