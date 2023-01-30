package batch

import (
	"fmt"
	"sync/atomic"
	"time"
)

// Stats can be used for counting processed and erroneous requests.
type Stats interface {
	AddProcessed()
	AddError()
	Values() (int, int)
}

// VerboseStats counts the processed and erroneous requests.
//
// VerboseStats is thread-safe with regards to incrementing the counter values.
type VerboseStats struct {
	processed *int64
	errors    *int64
}

// AddProcessed increases the processed requests counter by one.
func (s *VerboseStats) AddProcessed() {
	atomic.AddInt64(s.processed, 1)
}

// AddError increases the erroneous requests counter by one.
func (s *VerboseStats) AddError() {
	atomic.AddInt64(s.errors, 1)
}

// Values returns the current counter values (processed and erroneous requests).
//
// Values does not lock the counters while reading them, potentially resulting
// in situations where each counter value is correct at the time it was read,
// but not correct when looking at both values.
func (s *VerboseStats) Values() (int, int) {
	return int(*s.processed), int(*s.errors)
}

// SilentStats can be used when the statistics should not be collected.
type SilentStats struct{}

// AddProcessed is a no-op.
func (s *SilentStats) AddProcessed() {}

// AddError is a no-op.
func (s *SilentStats) AddError() {}

// Values always returns zero for the counter values.
func (s *SilentStats) Values() (int, int) {
	return 0, 0
}

// PrintStats creates an instance of Stats and if verbose is true starts
// printing these stats to stdout regularly.
//
// Currently there is no way to stop printing the stats.
func PrintStats(verbose bool, d time.Duration) Stats {
	if !verbose {
		return &SilentStats{}
	}

	s := &VerboseStats{
		processed: new(int64),
		errors:    new(int64),
	}
	start := time.Now()
	go func() {
		for {
			time.Sleep(d)
			processed, errors := s.Values()
			seconds := time.Since(start).Seconds()
			processedPerSecond := float64(processed) / seconds
			fmt.Printf("processed: %v (%.1f/s), errors: %v\n", processed, processedPerSecond, errors)
		}
	}()
	return s
}
