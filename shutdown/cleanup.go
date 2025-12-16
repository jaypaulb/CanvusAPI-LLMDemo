package shutdown

import (
	"context"
	"os"
	"path/filepath"

	"go_backend/core"

	"go.uber.org/zap"
)

// CleanupDownloads returns a shutdown function that removes temporary files
// from the downloads directory. It matches files with the "temp_*" pattern.
//
// Priority recommendation: 40+ (final cleanup, after services stopped)
//
// The cleanup function:
//   - Removes files matching "temp_*" in the downloads directory
//   - Logs each file removal (success or failure)
//   - Continues cleanup even if individual file removals fail
//   - Returns nil to avoid blocking shutdown (errors are logged)
//
// Usage:
//
//	manager.Register("cleanup-downloads", 45, shutdown.CleanupDownloads(logger, cfg.DownloadsDir))
func CleanupDownloads(logger *zap.Logger, downloadsDir string) core.ShutdownFunc {
	return func(ctx context.Context) error {
		return cleanupTempFiles(ctx, logger, downloadsDir)
	}
}

// CleanupDownloadsAndDir returns a shutdown function that removes all temporary
// files AND the downloads directory itself. Use this when the downloads directory
// is purely transient and should not persist between runs.
//
// Priority recommendation: 45+ (very final cleanup)
//
// Usage:
//
//	manager.Register("cleanup-downloads-dir", 50, shutdown.CleanupDownloadsAndDir(logger, cfg.DownloadsDir))
func CleanupDownloadsAndDir(logger *zap.Logger, downloadsDir string) core.ShutdownFunc {
	return func(ctx context.Context) error {
		// First clean up temp files
		if err := cleanupTempFiles(ctx, logger, downloadsDir); err != nil {
			// Log but continue - we still want to try removing the directory
			logger.Warn("Error during temp file cleanup, continuing with directory removal",
				zap.Error(err),
			)
		}

		// Check context before potentially expensive directory removal
		select {
		case <-ctx.Done():
			logger.Warn("Shutdown context cancelled, skipping directory removal")
			return nil
		default:
		}

		// Then remove the directory itself
		return removeDownloadsDir(logger, downloadsDir)
	}
}

// cleanupTempFiles removes files matching "temp_*" in the downloads directory.
// It returns nil even if some files fail to delete (errors are logged).
func cleanupTempFiles(ctx context.Context, logger *zap.Logger, downloadsDir string) error {
	logger.Debug("Starting temp file cleanup",
		zap.String("directory", downloadsDir),
	)

	pattern := filepath.Join(downloadsDir, "temp_*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		logger.Error("Failed to list temporary files",
			zap.String("pattern", pattern),
			zap.Error(err),
		)
		// Return nil to not block shutdown
		return nil
	}

	if len(matches) == 0 {
		logger.Debug("No temporary files to clean up")
		return nil
	}

	logger.Info("Cleaning up temporary files",
		zap.Int("file_count", len(matches)),
	)

	var removedCount int
	var failedCount int

	for _, match := range matches {
		// Check context between file deletions
		select {
		case <-ctx.Done():
			logger.Warn("Shutdown context cancelled during cleanup",
				zap.Int("removed", removedCount),
				zap.Int("remaining", len(matches)-removedCount-failedCount),
			)
			return nil
		default:
		}

		if err := os.Remove(match); err != nil {
			failedCount++
			logger.Warn("Failed to remove temporary file",
				zap.String("file", filepath.Base(match)),
				zap.Error(err),
			)
		} else {
			removedCount++
			logger.Debug("Removed temporary file",
				zap.String("file", filepath.Base(match)),
			)
		}
	}

	logger.Info("Temp file cleanup complete",
		zap.Int("removed", removedCount),
		zap.Int("failed", failedCount),
	)

	return nil
}

// removeDownloadsDir removes the downloads directory and all its contents.
// It returns nil if the directory doesn't exist.
func removeDownloadsDir(logger *zap.Logger, downloadsDir string) error {
	// Check if directory exists
	info, err := os.Stat(downloadsDir)
	if os.IsNotExist(err) {
		logger.Debug("Downloads directory does not exist, nothing to remove",
			zap.String("directory", downloadsDir),
		)
		return nil
	}
	if err != nil {
		logger.Error("Failed to stat downloads directory",
			zap.String("directory", downloadsDir),
			zap.Error(err),
		)
		// Return nil to not block shutdown
		return nil
	}

	if !info.IsDir() {
		logger.Warn("Downloads path is not a directory",
			zap.String("path", downloadsDir),
		)
		return nil
	}

	// Remove the directory and all contents
	if err := os.RemoveAll(downloadsDir); err != nil {
		logger.Error("Failed to remove downloads directory",
			zap.String("directory", downloadsDir),
			zap.Error(err),
		)
		// Return nil to not block shutdown
		return nil
	}

	logger.Info("Removed downloads directory",
		zap.String("directory", downloadsDir),
	)

	return nil
}
