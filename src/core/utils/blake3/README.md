# blake3 package

Derived from [zeebo/blake3](https://github.com/zeebo/blake3) originally released under [CC0 1.0](https://github.com/zeebo/blake3/blob/master/LICENSE) and [MIT](https://github.com/zeebo/blake3/blob/master/LICENSE-MIT).
Modified for this project's needs and requirements.

The port is based on upstream commit
`4f7123e6bac310831a1b0e8fbff6fabdc0ad37ec` (`v0.2.4-3-g4f7123e`).
The original license texts are preserved as `LICENSE-CC0` and `LICENSE-MIT`.

## Platform paths

- amd64: AVX2 tree hashing and SSE4.1 compression when supported by the CPU.
- arm64: NEON tree hashing when supported by the CPU.
- all other architectures: portable Go implementation.
- any architecture built with `-tags purego`: portable Go implementation.

The runtime CPU checks use `golang.org/x/sys/cpu`, which is already used by
this project. The upstream `cpuid` dependency is not included.

The generated SSE4.1, AVX2, and NEON assembly is committed so normal builds do
not require the upstream AVO code generator or its dependency graph. Builds
using the `purego` tag disable all assembly dispatch.

For troubleshooting or comparisons, individual optimized paths can also be
disabled at process startup with `BLAKE3_DISABLE_AVX2`,
`BLAKE3_DISABLE_SSE41`, or `BLAKE3_DISABLE_NEON`. Setting `BLAKE3_PUREGO`
disables every optimized path without rebuilding.

## Verification and benchmarks

```powershell
go test ./tests/blake3Tests
go test -tags purego ./tests/blake3Tests
go test -run '^$' -bench '^BenchmarkChunkDigest$' -benchmem ./tests/blake3Tests
go test -run '^$' -bench '^BenchmarkChunkDigest$' -benchmem -tags purego ./tests/blake3Tests
```

The tests include the official unkeyed, keyed, key-derivation, and
extendable-output vectors, a large-input regression vector, and direct
optimized-versus-portable implementation comparisons.
