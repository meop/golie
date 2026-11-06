package running

import "golang.org/x/sys/windows"

// marked for deletion and unmarked again: the marking is refused for a running program's file, and is the question
func held(path string) bool {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	h, err := windows.CreateFile(p, windows.DELETE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return false
	}
	defer func() { _ = windows.CloseHandle(h) }()
	deleteFile := byte(1)
	if err := windows.SetFileInformationByHandle(h, windows.FileDispositionInfo, &deleteFile, 1); err != nil {
		return err == windows.ERROR_ACCESS_DENIED
	}
	deleteFile = 0
	_ = windows.SetFileInformationByHandle(h, windows.FileDispositionInfo, &deleteFile, 1)
	return false
}
