//go:build !linux && !darwin && !windows

package running

func programs() []string { return nil }
