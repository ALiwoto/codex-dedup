package localProxy

import (
	"errors"
	"io"
)

// DedupMeasurementSnapshot returns a concurrency-safe copy of cumulative measurements.
func (p *LocalProxy) DedupMeasurementSnapshot() DedupMeasurementSnapshot {
	return p.measurementStore.getSnapshot()
}

func (s *dedupMeasurementStore) recordRequest(
	bodyBytes uint64,
	chunks []chunkIdentity,
) requestDedupMeasurement {
	s.mut.Lock()
	defer s.mut.Unlock()

	measurement := requestDedupMeasurement{
		BodyBytes:  bodyBytes,
		ChunkCount: uint64(len(chunks)),
	}
	for _, chunk := range chunks {
		chunkBytes := uint64(chunk.length)
		if _, exists := s.chunks[chunk]; exists {
			measurement.ReusableChunkBytes += chunkBytes
			measurement.ReusableChunkCount++
			continue
		}

		s.chunks[chunk] = struct{}{}
		measurement.NewChunkBytes += chunkBytes
		measurement.NewChunkCount++
		s.snapshot.SimulatedCacheBytes += chunkBytes
	}

	measurement.SimulatedCacheChunks = uint64(len(s.chunks))
	measurement.SimulatedCacheBytes = s.snapshot.SimulatedCacheBytes
	s.snapshot.MeasuredRequestCount++
	s.snapshot.BodyBytes += measurement.BodyBytes
	s.snapshot.ChunkCount += measurement.ChunkCount
	s.snapshot.NewChunkBytes += measurement.NewChunkBytes
	s.snapshot.ReusableChunkBytes += measurement.ReusableChunkBytes
	s.snapshot.NewChunkCount += measurement.NewChunkCount
	s.snapshot.ReusableChunkCount += measurement.ReusableChunkCount
	s.snapshot.SimulatedCacheChunks = measurement.SimulatedCacheChunks

	return measurement
}

func (s *dedupMeasurementStore) getSnapshot() DedupMeasurementSnapshot {
	s.mut.Lock()
	defer s.mut.Unlock()

	return s.snapshot
}

func (s *measuredBodyStream) Read(output []byte) (int, error) {
	readCount, err := s.source.Read(output)
	s.readBytes += int64(readCount)
	if readCount != 0 && !s.measurementWriteFailed {
		if _, writeErr := s.measurementWriter.Write(output[:readCount]); writeErr != nil {
			s.measurementWriteFailed = true
		}
	}

	if err != nil || s.hasCompleteBody(err) {
		s.closeMeasurement.Do(func() {
			if s.hasCompleteBody(err) {
				_ = s.measurementWriter.Close()
				return
			}
			if err != nil && !errors.Is(err, io.EOF) {
				_ = s.measurementWriter.CloseWithError(err)
				return
			}
			_ = s.measurementWriter.CloseWithError(io.ErrUnexpectedEOF)
		})
	}

	return readCount, err
}

func (s *measuredBodyStream) Close() error {
	s.closeMeasurement.Do(func() {
		if s.hasCompleteBody(nil) {
			_ = s.measurementWriter.Close()
			return
		}
		_ = s.measurementWriter.CloseWithError(io.ErrUnexpectedEOF)
	})
	return nil
}

func (s *measuredBodyStream) hasCompleteBody(readErr error) bool {
	if s.expectedBytes >= 0 {
		return s.readBytes == s.expectedBytes
	}
	return errors.Is(readErr, io.EOF)
}
