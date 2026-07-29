package fastCdc

const (
	AlgorithmVersion = "fastcdc-v1"

	DefaultMinimumSize = 16 * 1024
	DefaultAverageSize = 64 * 1024
	DefaultMaximumSize = 256 * 1024

	normalizationLevel = 2
	gearTableSeed      = uint64(0x6a09e667f3bcc909)
	splitMixIncrement  = uint64(0x9e3779b97f4a7c15)
	splitMixFactor1    = uint64(0xbf58476d1ce4e5b9)
	splitMixFactor2    = uint64(0x94d049bb133111eb)
)
