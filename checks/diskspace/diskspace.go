package diskspace

import (
	"context"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
)

// New creates a new Diskspace health check function
func New(thresholdBytes uint64, directory string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		// Query the OS about filesystem
		var stat unix.Statfs_t
		err := unix.Statfs(directory, &stat)
		if err != nil {
			return err
		}
		// Get information from OS query
		blockSize := uint64(stat.Bsize)
		total := stat.Blocks * blockSize
		available := stat.Bavail * blockSize
		// Evaluate filessystem state and return any errors
		return check(ctx, thresholdBytes, total, available)
	}
}

// Wrapper function around New() to provide default working directory
func NewWorkingDirectory(thresholdBytes uint64) func(ctx context.Context) error {
	wd, err := os.Getwd()
	if err != nil {
		panic("Unable to get current working directory, cannot return new check")
	}
	return New(thresholdBytes, wd)
}

// Perform logical check on provided values
func check(_ context.Context, thresholdBytes uint64, totalBytes uint64, availableBytes uint64) error {
	usedBytes := totalBytes - availableBytes
	if usedBytes > thresholdBytes {
		return fmt.Errorf("Disk usage has exceeded the specified threshold of %d bytes and is now %d bytes", thresholdBytes, usedBytes)
	}
	return nil
}
