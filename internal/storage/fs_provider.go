package storage

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
)

// FSProvider implements Provider using the local filesystem.
type FSProvider struct {
	baseDir string
	logger  *slog.Logger
}

var validAgentIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// NewFSProvider creates a new filesystem-based storage provider.
// The baseDir will be created if it doesn't exist.
func NewFSProvider(baseDir string, logger *slog.Logger) (*FSProvider, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	logger.Info("Filesystem storage provider created", "base_dir", baseDir)

	return &FSProvider{
		baseDir: baseDir,
		logger:  logger,
	}, nil
}

// SaveCheckpoint saves an agent checkpoint atomically.
// Uses atomic write pattern: temp file -> fsync -> rename.
func (p *FSProvider) SaveCheckpoint(
	ctx context.Context,
	agentID string,
	state []byte,
) error {
	checkpointPath, pathErr := p.checkpointPath(agentID)
	if pathErr != nil {
		return pathErr
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
	checkpointPath, pathErr := p.checkpointPath(agentID)
	if pathErr != nil {
		return nil, pathErr
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
	checkpointPath, pathErr := p.checkpointPath(agentID)
	if pathErr != nil {
		return pathErr
	}

	err := os.Remove(checkpointPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	p.logger.Info("Checkpoint deleted", "agent_id", agentID, "path", checkpointPath)
	return nil
}

func validateAgentID(agentID string) error {
	if !validAgentIDPattern.MatchString(agentID) {
		return fmt.Errorf("invalid agent_id %q", agentID)
	}
	return nil
}

// checkpointPath returns the filesystem path for an agent's checkpoint.
func (p *FSProvider) checkpointPath(agentID string) (string, error) {
	if err := validateAgentID(agentID); err != nil {
		return "", err
	}

	return filepath.Join(p.baseDir, agentID+".checkpoint"), nil
}
