package localProxy

import (
	"io"
	"net/url"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

type LocalProxyOptions struct {
	BindAddress          string
	ProviderURL          *url.URL
	LogDedupMeasurements bool
}

type LocalProxy struct {
	app               *fiber.App
	bindAddress       string
	providerHost      string
	providerURLPrefix string
	providerClient    *fasthttp.Client
	measurementStore  *dedupMeasurementStore
	logMeasurements   bool
}

type mutableHTTPHeader interface {
	Peek(key string) []byte
	Del(key string)
}

type chunkIdentity struct {
	digest [32]byte
	length int
}

// DedupMeasurementSnapshot contains content-free cumulative local measurements.
type DedupMeasurementSnapshot struct {
	MeasuredRequestCount uint64
	BodyBytes            uint64
	ChunkCount           uint64
	NewChunkBytes        uint64
	ReusableChunkBytes   uint64
	NewChunkCount        uint64
	ReusableChunkCount   uint64
	SimulatedCacheBytes  uint64
	SimulatedCacheChunks uint64
}

type requestDedupMeasurement struct {
	BodyBytes            uint64
	ChunkCount           uint64
	NewChunkBytes        uint64
	ReusableChunkBytes   uint64
	NewChunkCount        uint64
	ReusableChunkCount   uint64
	SimulatedCacheBytes  uint64
	SimulatedCacheChunks uint64
}

type dedupMeasurementStore struct {
	mut      sync.Mutex
	chunks   map[chunkIdentity]struct{}
	snapshot DedupMeasurementSnapshot
}

type bodyMeasurementResult struct {
	measurement requestDedupMeasurement
	err         error
}

type measuredBodyStream struct {
	source                 io.Reader
	measurementWriter      *io.PipeWriter
	closeMeasurement       sync.Once
	expectedBytes          int64
	readBytes              int64
	measurementWriteFailed bool
}
