package util

const metricsBufferSize = 256

// MetricsGetter allows access to tracked performance metrics.
type MetricsGetter interface {
	Avg() uint64
	GetLast() uint64
}

// MetricsHandler takes care of storing and updating tracked performance
// metrics and computes a rolling average to give basic insight into program
// performance.
type MetricsHandler struct {
	values [metricsBufferSize]uint64
	index  uint64
}

// GetLast returns the most recently added performance value.
func (h *MetricsHandler) GetLast() uint64 {
	return h.values[h.index]
}

// Add inserts a new performance value into the ring buffer.
func (h *MetricsHandler) Add(value uint64) {
	h.index = (h.index + 1) % metricsBufferSize
	h.values[h.index] = value
}

// Avg returns the average from the ring buffer.
// Note that this can be misleading before the ring buffer has been filled up.
func (h *MetricsHandler) Avg() uint64 {
	sum := uint64(0)
	for _, v := range h.values {
		sum += v
	}
	return (sum / metricsBufferSize)
}
