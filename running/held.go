package running

// Held reports whether the OS will not delete the file at path because a
// program runs from it. Only Windows holds one: it lets a running program's
// file be renamed, which is how a shim stamped over one is set aside, but not
// deleted, and it names the process by the path it started from, not the
// name the file has now.
func Held(path string) bool { return held(path) }
