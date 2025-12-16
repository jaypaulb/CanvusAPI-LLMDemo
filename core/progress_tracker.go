package core

import (
	"sync"
	"time"
)

// ProgressInfo contains the current download progress information.
// This is returned by ProgressTracker.Progress() for display.
type ProgressInfo struct {
	// Total bytes to download (0 if unknown)
	Total int64
	// Downloaded bytes so far
	Downloaded int64
	// Percentage complete (0-100, or -1 if total is unknown)
	Percent float64
	// Download speed in bytes per second
	SpeedBytesPerSec float64
	// Speed formatted as human-readable string (e.g., "5.2 MB/s")
	SpeedFormatted string
	// Estimated time remaining (0 if unknown or complete)
	ETA time.Duration
	// Elapsed time since download started
	Elapsed time.Duration
	// Human-readable downloaded size
	DownloadedFormatted string
	// Human-readable total size (or "unknown" if 0)
	TotalFormatted string
}

// ProgressTracker tracks download progress with thread-safe updates.
// It calculates speed, ETA, and provides formatted progress information.
type ProgressTracker struct {
	mu sync.RWMutex

	// Total bytes to download (0 if unknown)
	total int64
	// Downloaded bytes so far
	downloaded int64
	// Time when download started
	startTime time.Time
	// Last update time for speed calculation
	lastUpdateTime time.Time
	// Bytes downloaded at last update (for speed calculation)
	lastDownloaded int64
	// Moving average of speed (bytes/sec)
	speedAvg float64
	// Weight for exponential moving average (0-1, higher = more recent data)
	speedAlpha float64
}

// NewProgressTracker creates a new progress tracker.
// total is the total bytes to download (use 0 if unknown).
func NewProgressTracker(total int64) *ProgressTracker {
	now := time.Now()
	return &ProgressTracker{
		total:          total,
		downloaded:     0,
		startTime:      now,
		lastUpdateTime: now,
		lastDownloaded: 0,
		speedAvg:       0,
		speedAlpha:     0.3, // Balance between responsiveness and smoothness
	}
}

// Update adds n bytes to the downloaded count.
// This method is thread-safe.
func (p *ProgressTracker) Update(n int64) {
	if n <= 0 {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.downloaded += n
	p.updateSpeed()
}

// SetDownloaded sets the absolute downloaded byte count.
// This method is thread-safe.
func (p *ProgressTracker) SetDownloaded(downloaded int64) {
	if downloaded < 0 {
		downloaded = 0
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.downloaded = downloaded
	p.updateSpeed()
}

// SetTotal updates the total bytes to download.
// This method is thread-safe.
func (p *ProgressTracker) SetTotal(total int64) {
	if total < 0 {
		total = 0
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.total = total
}

// updateSpeed recalculates the download speed.
// Must be called with mu held.
func (p *ProgressTracker) updateSpeed() {
	now := time.Now()
	elapsed := now.Sub(p.lastUpdateTime).Seconds()

	// Only update speed if some time has passed
	if elapsed >= 0.1 {
		bytesInInterval := p.downloaded - p.lastDownloaded
		instantSpeed := float64(bytesInInterval) / elapsed

		// Exponential moving average for smooth speed display
		if p.speedAvg == 0 {
			p.speedAvg = instantSpeed
		} else {
			p.speedAvg = p.speedAlpha*instantSpeed + (1-p.speedAlpha)*p.speedAvg
		}

		p.lastUpdateTime = now
		p.lastDownloaded = p.downloaded
	}
}

// Progress returns the current progress information.
// This method is thread-safe.
func (p *ProgressTracker) Progress() ProgressInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	now := time.Now()
	elapsed := now.Sub(p.startTime)

	info := ProgressInfo{
		Total:               p.total,
		Downloaded:          p.downloaded,
		Percent:             -1, // Unknown if total is 0
		SpeedBytesPerSec:    p.speedAvg,
		SpeedFormatted:      FormatBytes(int64(p.speedAvg)) + "/s",
		ETA:                 0,
		Elapsed:             elapsed,
		DownloadedFormatted: FormatBytes(p.downloaded),
		TotalFormatted:      "unknown",
	}

	// Calculate percentage if total is known
	if p.total > 0 {
		info.Percent = float64(p.downloaded) / float64(p.total) * 100
		info.TotalFormatted = FormatBytes(p.total)

		// Cap percentage at 100
		if info.Percent > 100 {
			info.Percent = 100
		}

		// Calculate ETA if we have speed and not yet complete
		if p.speedAvg > 0 && p.downloaded < p.total {
			remaining := float64(p.total - p.downloaded)
			etaSeconds := remaining / p.speedAvg
			info.ETA = time.Duration(etaSeconds * float64(time.Second))
		}
	}

	return info
}

// Downloaded returns the current downloaded byte count.
// This method is thread-safe.
func (p *ProgressTracker) Downloaded() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.downloaded
}

// Total returns the total bytes to download.
// This method is thread-safe.
func (p *ProgressTracker) Total() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.total
}

// IsComplete returns true if download is complete (downloaded >= total).
// Returns false if total is unknown (0).
// This method is thread-safe.
func (p *ProgressTracker) IsComplete() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.total > 0 && p.downloaded >= p.total
}

// Reset resets the tracker to start a new download.
// This method is thread-safe.
func (p *ProgressTracker) Reset(total int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	p.total = total
	p.downloaded = 0
	p.startTime = now
	p.lastUpdateTime = now
	p.lastDownloaded = 0
	p.speedAvg = 0
}
