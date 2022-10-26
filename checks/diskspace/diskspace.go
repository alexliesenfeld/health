package diskspace

import (
	"context"
	"fmt"
	"syscall"
)

// New creates a new Diskspace health check function
func New(thresholdBytes uint64, directory string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		// Query the OS about filesystem
		var stat syscall.Statfs_t
		err := syscall.Statfs(directory, &stat)
		if err != nil {
			return err
		}
		// Get information from OS query
		blockSize := uint64(stat.Bsize)
		totalBytes := stat.Blocks * blockSize
		availableBytes := stat.Bavail * blockSize
		// Evaluate filessystem state and return any errors
		usedBytes := totalBytes - availableBytes
		if usedBytes > thresholdBytes {
			return fmt.Errorf("Disk usage has exceeded the specified threshold of %d bytes and is now %d bytes", thresholdBytes, usedBytes)
		}
		return nil
	}
}
