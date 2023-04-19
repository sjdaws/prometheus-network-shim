package cri

import (
	"runtime"
	"time"
)

const defaultTimeout = 2 * time.Second
const defaultTimeoutWindows = 200 * time.Second

// getTimeout returns the timeout to use for crictl
func getTimeout(timeout int) time.Duration {
	if timeout > 0 {
		return time.Duration(timeout) * time.Second
	}

	if runtime.GOOS == "windows" {
		return defaultTimeoutWindows
	}

	return defaultTimeout
}
