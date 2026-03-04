package storage

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// FSProvider implements Provider using the local filesystem.
type FSProvider struct {
	baseDir string
	logger  *slog.Logger
}

// NewFSProvider creates a new filesystem-based storage provider.
// The baseDir will be created if it doesn't exist.
func NewFSProvider(baseDir string, logger *slog.Logger) (*FSProvider, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	logger.Info("Filesystem storage provider created", "base_dir", baseDir)

	p := &FSProvider{
		baseDir: baseDir,
		logger:  logger,
	}
	p.cleanStaleTempFiles()
	return p, nil
}

// cleanStaleTempFiles removes leftover .tmp files from interrupted checkpoints.
// A crash between fsync and rename leaves a .tmp file that should be discarded
// to avoid promoting a partial write on the next checkpoint attempt.
func (p *FSProvider) cleanStaleTempFiles() {
	pattern := filepath.Join(p.baseDir, "*.tmp")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		p.logger.Warn("Failed to glob for stale temp files", "error", err)
		return
	}
	for _, m := range matches {
		if err := os.Remove(m); err != nil {
			p.logger.Warn("Failed to remove stale temp file", "path", m, "error", err)
		} else {
			p.logger.Info("Removed stale temp file", "path", m)
		}
	}
}

// SaveCheckpoint saves an agent checkpoint atomically.
// Uses atomic write pattern: temp file -> fsync -> rename.
func (p *FSProvider) SaveCheckpoint(
	ctx context.Context,
	agentID string,
	state []byte,
) error {
	checkpointPath, err := p.checkpointPath(agentID)
	if err != nil {
		return err
	}
	tempPath := checkpointPath + ".tmp"

	// Write to temporary file
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp checkpoint: %w", err)
	}

	// Ensure temp file is cleaned up on error
	defer func() {
		if err != nil {
			os.Remove(tempPath)
		}
	}()

	// Write state data
	if _, err = tempFile.Write(state); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write checkpoint: %w", err)
	}

	// Fsync to ensure data is on disk
	if err = tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to fsync checkpoint: %w", err)
	}

	// Close before rename
	if err = tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp checkpoint: %w", err)
	}

	// Atomic rename
	if err = os.Rename(tempPath, checkpointPath); err != nil {
		return fmt.Errorf("failed to rename checkpoint: %w", err)
	}

	p.logger.Info("Checkpoint saved",
		"agent_id", agentID,
		"path", checkpointPath,
		"size_bytes", len(state),
	)

	return nil
}

// LoadCheckpoint loads an agent checkpoint from disk.
func (p *FSProvider) LoadCheckpoint(
	ctx context.Context,
	agentID string,
) ([]byte, error) {
	checkpointPath, err := p.checkpointPath(agentID)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(checkpointPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrCheckpointNotFound
		}
		return nil, fmt.Errorf("failed to read checkpoint: %w", err)
	}

	p.logger.Info("Checkpoint loaded",
		"agent_id", agentID,
		"path", checkpointPath,
		"size_bytes", len(data),
	)

	return data, nil
}

// DeleteCheckpoint removes an agent checkpoint from disk.
func (p *FSProvider) DeleteCheckpoint(
	ctx context.Context,
	agentID string,
) error {
	checkpointPath, err := p.checkpointPath(agentID)
	if err != nil {
		return err
	}

	err = os.Remove(checkpointPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	p.logger.Info("Checkpoint deleted", "agent_id", agentID, "path", checkpointPath)
	return nil
}

// SaveReceipts persists an agent's serialized receipt chain atomically.
func (p *FSProvider) SaveReceipts(
	ctx context.Context,
	agentID string,
	data []byte,
) error {
	rPath, err := p.receiptPath(agentID)
	if err != nil {
		return err
	}
	tempPath := rPath + ".tmp"

	tempFile, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp receipts: %w", err)
	}

	defer func() {
		if err != nil {
			os.Remove(tempPath)
		}
	}()

	if _, err = tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write receipts: %w", err)
	}

	if err = tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to fsync receipts: %w", err)
	}

	if err = tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp receipts: %w", err)
	}

	if err = os.Rename(tempPath, rPath); err != nil {
		return fmt.Errorf("failed to rename receipts: %w", err)
	}

	p.logger.Info("Receipts saved",
		"agent_id", agentID,
		"path", rPath,
		"size_bytes", len(data),
	)
	return nil
}

// LoadReceipts retrieves an agent's serialized receipt chain.
func (p *FSProvider) LoadReceipts(
	ctx context.Context,
	agentID string,
) ([]byte, error) {
	rPath, err := p.receiptPath(agentID)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(rPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrReceiptsNotFound
		}
		return nil, fmt.Errorf("failed to read receipts: %w", err)
	}

	p.logger.Info("Receipts loaded",
		"agent_id", agentID,
		"path", rPath,
		"size_bytes", len(data),
	)
	return data, nil
}

// DeleteReceipts removes an agent's receipt chain.
func (p *FSProvider) DeleteReceipts(
	ctx context.Context,
	agentID string,
) error {
	rPath, err := p.receiptPath(agentID)
	if err != nil {
		return err
	}

	err = os.Remove(rPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete receipts: %w", err)
	}

	p.logger.Info("Receipts deleted", "agent_id", agentID, "path", rPath)
	return nil
}

// checkpointPath returns the filesystem path for an agent's checkpoint.
// Returns an error if the agentID would escape the base directory (path traversal).
func (p *FSProvider) checkpointPath(agentID string) (string, error) {
	path := filepath.Join(p.baseDir, agentID+".checkpoint")
	cleaned := filepath.Clean(path)
	if !strings.HasPrefix(cleaned, filepath.Clean(p.baseDir)+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid agent ID: path traversal detected")
	}
	return cleaned, nil
}

// receiptPath returns the filesystem path for an agent's receipts.
func (p *FSProvider) receiptPath(agentID string) (string, error) {
	path := filepath.Join(p.baseDir, agentID+".receipts")
	cleaned := filepath.Clean(path)
	if !strings.HasPrefix(cleaned, filepath.Clean(p.baseDir)+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid agent ID: path traversal detected")
	}
	return cleaned, nil
}
