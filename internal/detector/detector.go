// Package detector provides peer detection functionality
package detector

import (
	"sync"
	"time"

	"github.com/lbls/aria2bango/internal/aria2"
	"github.com/lbls/aria2bango/internal/config"
)

// DetectionResult represents a detection result
type DetectionResult struct {
	Peer          aria2.Peer
	Reason        string
	ShareRatio    float64
	Violations    int           // 违规次数
	BlockDuration time.Duration // 本次屏蔽时长
}

// Detector handles peer detection
type Detector struct {
	config     *config.DetectionConfig
	peerStats  map[string]*PeerStats
	statsMutex sync.RWMutex
}

// PeerStats tracks peer statistics for behavior analysis
type PeerStats struct {
	IP            string
	TotalDownload int64
	TotalUpload   int64
	FirstSeen     time.Time
	LastSeen      time.Time
	Violations    int       // 违规次数（累加惩罚）
	LastBlocked   time.Time // 上次屏蔽时间
}

// NewDetector creates a new detector
func NewDetector(cfg *config.DetectionConfig) *Detector {
	return &Detector{
		config:    cfg,
		peerStats: make(map[string]*PeerStats),
	}
}

// Detect checks if a peer is a leecher based on behavior analysis
func (d *Detector) Detect(peer aria2.Peer, baseBlockDuration time.Duration) *DetectionResult {
	// Only use behavior analysis
	if d.config.Behavior.Enabled {
		return d.analyzeBehavior(peer, baseBlockDuration)
	}
	return nil
}

// analyzeBehavior checks if peer exhibits leeching behavior
func (d *Detector) analyzeBehavior(peer aria2.Peer, baseBlockDuration time.Duration) *DetectionResult {
	d.statsMutex.Lock()
	defer d.statsMutex.Unlock()

	stats, exists := d.peerStats[peer.IP]
	if !exists {
		stats = &PeerStats{
			IP:        peer.IP,
			FirstSeen: time.Now(),
		}
		d.peerStats[peer.IP] = stats
	}

	// Update statistics
	// Note: aria2's downloadSpeed = speed we download FROM peer (peer uploads to us)
	//       aria2's uploadSpeed = speed we upload TO peer (peer downloads from us)
	// So from peer's perspective:
	//   - peer's upload = our download from them
	//   - peer's download = our upload to them
	stats.TotalDownload += peer.DownloadSpeed // peer's upload (what they give us)
	stats.TotalUpload += peer.UploadSpeed     // peer's download (what they take from us)
	stats.LastSeen = time.Now()

	// Check if we have enough data
	if stats.TotalUpload < d.config.Behavior.MinDataThreshold {
		return nil
	}

	// Calculate share ratio from peer's perspective
	// shareRatio = peer's upload / peer's download
	// A leecher has low shareRatio (uploads little, downloads a lot)
	shareRatio := float64(0)
	if stats.TotalUpload > 0 {
		shareRatio = float64(stats.TotalDownload) / float64(stats.TotalUpload)
	}

	// Check if share ratio is below threshold
	// Low shareRatio means peer downloads a lot but uploads little
	if shareRatio < d.config.Behavior.MinShareRatio {
		// Increment violation count
		stats.Violations++

		// Calculate block duration: violations * base_duration
		// e.g., 1st: 1*5min, 2nd: 2*5min, 3rd: 3*5min
		blockDuration := time.Duration(stats.Violations) * baseBlockDuration
		stats.LastBlocked = time.Now()

		return &DetectionResult{
			Peer:          peer,
			Reason:        "low_share_ratio",
			ShareRatio:    shareRatio,
			Violations:    stats.Violations,
			BlockDuration: blockDuration,
		}
	}

	return nil
}

// GetViolationCount returns the violation count for an IP
func (d *Detector) GetViolationCount(ip string) int {
	d.statsMutex.RLock()
	defer d.statsMutex.RUnlock()
	if stats, exists := d.peerStats[ip]; exists {
		return stats.Violations
	}
	return 0
}

// ResetViolations resets the violation count for an IP
func (d *Detector) ResetViolations(ip string) {
	d.statsMutex.Lock()
	defer d.statsMutex.Unlock()
	if stats, exists := d.peerStats[ip]; exists {
		stats.Violations = 0
	}
}

// CleanupStaleStats removes stale peer statistics
func (d *Detector) CleanupStaleStats(maxAge time.Duration) {
	d.statsMutex.Lock()
	defer d.statsMutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for ip, stats := range d.peerStats {
		if stats.LastSeen.Before(cutoff) {
			delete(d.peerStats, ip)
		}
	}
}

// GetStats returns current peer statistics
func (d *Detector) GetStats(ip string) *PeerStats {
	d.statsMutex.RLock()
	defer d.statsMutex.RUnlock()
	return d.peerStats[ip]
}

// GetAllStats returns all peer statistics
func (d *Detector) GetAllStats() map[string]*PeerStats {
	d.statsMutex.RLock()
	defer d.statsMutex.RUnlock()
	result := make(map[string]*PeerStats, len(d.peerStats))
	for k, v := range d.peerStats {
		result[k] = v
	}
	return result
}
