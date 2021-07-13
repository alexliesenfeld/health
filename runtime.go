package health

import "runtime"

type (
	runtimeSnapshot struct {
		Version         string `json:"version,omitempty"`
		GoroutineCount  int    `json:"goroutineCount,omitempty"`
		TotalAllocBytes int    `json:"totalAllocBytes,omitempty"`
		HeapObjectCount int    `json:"heapObjectCount,omitempty"`
		AllocBytes      int    `json:"allocBytes,omitempty"`
	}
)

func newRuntimeSnapshot() runtimeSnapshot {
	s := runtime.MemStats{}
	runtime.ReadMemStats(&s)

	return runtimeSnapshot{
		Version:         runtime.Version(),
		GoroutineCount:  runtime.NumGoroutine(),
		TotalAllocBytes: int(s.TotalAlloc),
		HeapObjectCount: int(s.HeapObjects),
		AllocBytes:      int(s.Alloc),
	}
}
