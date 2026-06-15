package clone

import (
	"sync"
	"sync/atomic"
)

// maxRecordedFailures caps how many individual failures Run keeps for the final
// report, so a huge broken site cannot grow the slice without bound. The error
// counters still count every failure.
const maxRecordedFailures = 100

// stats are the live counters of a run, read by the CLI's progress ticker.
type stats struct {
	pages       atomic.Int64
	assets      atomic.Int64
	pageErrors  atomic.Int64
	assetErrors atomic.Int64
	skipped     atomic.Int64 // robots-disallowed or out of budget

	muFail   sync.Mutex
	failures []Failure
}

// Failure is one thing that went wrong, kept for the end-of-run report so the
// errors are visible as a list rather than only as a count.
type Failure struct {
	Kind    string // "page" or "asset"
	URL     string
	Referer string // the page that referenced it, when known
	Reason  string // e.g. "HTTP 403 Forbidden"
}

func (s *stats) recordFailure(f Failure) {
	s.muFail.Lock()
	if len(s.failures) < maxRecordedFailures {
		s.failures = append(s.failures, f)
	}
	s.muFail.Unlock()
}

func (s *stats) recordedFailures() []Failure {
	s.muFail.Lock()
	defer s.muFail.Unlock()
	out := make([]Failure, len(s.failures))
	copy(out, s.failures)
	return out
}

// Progress is a snapshot of a run for display.
type Progress struct {
	Pages       int64
	Assets      int64
	PageErrors  int64
	AssetErrors int64
	Skipped     int64
}

func (s *stats) snapshot() Progress {
	return Progress{
		Pages:       s.pages.Load(),
		Assets:      s.assets.Load(),
		PageErrors:  s.pageErrors.Load(),
		AssetErrors: s.assetErrors.Load(),
		Skipped:     s.skipped.Load(),
	}
}

// Result is the final outcome returned by Run.
type Result struct {
	Progress
	OutDir string
	// Failures is a capped sample of what went wrong, for the final report.
	Failures []Failure
}
