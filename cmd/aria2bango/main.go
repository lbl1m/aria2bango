// Package main is the entry point for aria2bango
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/lbl1m/aria2bango/internal/aria2"
	"github.com/lbl1m/aria2bango/internal/config"
	"github.com/lbl1m/aria2bango/internal/detector"
	"github.com/lbl1m/aria2bango/internal/firewall"
	"github.com/lbl1m/aria2bango/internal/logger"
	"github.com/lbl1m/aria2bango/internal/peerid"
)

var (
	configPath  = flag.String("config", "/etc/aria2bango/config.yaml", "Path to configuration file")
	cleanupMode = flag.Bool("cleanup", false, "Cleanup nftables rules and exit")
	version     = "dev"
)

func main() {
	flag.Parse()

	// Setup logger
	zapConfig := zap.NewProductionConfig()
	zapConfig.EncoderConfig.TimeKey = "timestamp"
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapLogger, err := zapConfig.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer zapLogger.Sync()
	log := zapLogger.Sugar()

	// Cleanup mode
	if *cleanupMode {
		log.Info("Running in cleanup mode, removing nftables rules...")
		if err := firewall.Cleanup("aria2bango"); err != nil {
			log.Errorf("Failed to cleanup nftables: %v", err)
			os.Exit(1)
		}
		log.Info("Cleanup completed successfully")
		return
	}

	// Load configuration
	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Warnf("Failed to load config from %s, using defaults: %v", *configPath, err)
		cfg = config.DefaultConfig()
	}

	// Initialize components
	aria2Client := aria2.NewClient(cfg.Aria2.Host, cfg.Aria2.Port, cfg.Aria2.Secret)
	det := detector.NewDetector(&cfg.Detection)

	// Initialize nftables manager
	nftMgr, err := firewall.NewNftablesManager(cfg.Blocking.NftTable)
	if err != nil {
		log.Fatalf("Failed to initialize nftables: %v", err)
	}
	defer func() {
		log.Info("Cleaning up nftables rules...")
		if err := nftMgr.Destroy(); err != nil {
			log.Errorf("Failed to cleanup nftables: %v", err)
		}
		nftMgr.Close()
	}()

	// Initialize logger
	blockLogger, err := logger.NewLogger(&cfg.Logging)
	if err != nil {
		log.Fatalf("Failed to initialize block logger: %v", err)
	}
	defer blockLogger.Close()

	log.Infof("aria2bango %s started", version)
	log.Infof("Monitoring aria2 at %s:%d", cfg.Aria2.Host, cfg.Aria2.Port)
	log.Infof("Base block duration: %s (cumulative punishment enabled)", cfg.Blocking.BaseDuration)

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Infof("Received signal %v, shutting down...", sig)
		cancel()
	}()

	// Main monitoring loop
	ticker := time.NewTicker(cfg.Aria2.PollInterval)
	defer ticker.Stop()

	// Periodic cleanup of stale peer stats
	cleanupTicker := time.NewTicker(5 * time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Shutting down...")
			return

		case <-cleanupTicker.C:
			det.CleanupStaleStats(30 * time.Minute)

		case <-ticker.C:
			if err := monitorPeers(ctx, aria2Client, det, nftMgr, blockLogger, cfg, log); err != nil {
				log.Errorf("Error monitoring peers: %v", err)
			}
		}
	}
}

func loadConfig(path string) (*config.Config, error) {
	// Check if config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	return config.Load(path)
}

func monitorPeers(ctx context.Context, aria2Client *aria2.Client, det *detector.Detector, nftMgr *firewall.NftablesManager, blockLogger *logger.Logger, cfg *config.Config, log *zap.SugaredLogger) error {
	// Get all peers from active downloads
	allPeers, err := aria2Client.GetAllPeers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get peers: %w", err)
	}

	// Check each peer
	for gid, peers := range allPeers {
		for _, peer := range peers {
			// Detect leecher behavior, pass base duration for cumulative punishment
			result := det.Detect(peer, cfg.Blocking.BaseDuration)
			if result == nil {
				continue
			}

			// Block the peer with calculated duration (violations * base_duration)
			if err := nftMgr.BlockIP(peer.IP, result.BlockDuration); err != nil {
				log.Errorf("Failed to block IP %s: %v", peer.IP, err)
				continue
			}

			log.Infof("Blocked %s (reason: %s, violations: %d, duration: %s, share_ratio: %.4f)",
				peer.IP, result.Reason, result.Violations, result.BlockDuration, result.ShareRatio)

			// Log the block event
			if err := blockLogger.LogBlock(logger.BlockEvent{
				IP:            peer.IP,
				PeerID:        peer.PeerID,
				ClientName:    peerid.GetNameWithVersion(peer.PeerID),
				Reason:        result.Reason,
				Duration:      result.BlockDuration.String(),
				DownloadSpeed: peer.DownloadSpeed,
				UploadSpeed:   peer.UploadSpeed,
				ShareRatio:    result.ShareRatio,
			}); err != nil {
				log.Errorf("Failed to log block event: %v", err)
			}

			// Log gid for debugging
			log.Debugf("Blocked peer from download %s", gid)
		}
	}

	return nil
}
