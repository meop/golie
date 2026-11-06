//go:build !windows

package running

func held(string) bool { return false }
