//go:build windows

package link

import (
	"encoding/binary"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func replace(path, dir string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	return create(path, dir)
}

// create makes path a junction to dir, as mklink /J does
func create(path, dir string) error {
	if err := os.Mkdir(path, 0755); err != nil {
		return err
	}
	if err := mount(path, dir); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func mount(path, dir string) error {
	substitute, err := windows.UTF16FromString(`\??\` + dir)
	if err != nil {
		return err
	}
	print, err := windows.UTF16FromString(dir)
	if err != nil {
		return err
	}
	names := append(substitute, print...)
	// REPARSE_DATA_BUFFER's MountPointReparseBuffer
	head := make([]byte, 16)
	binary.LittleEndian.PutUint32(head[0:], windows.IO_REPARSE_TAG_MOUNT_POINT)
	binary.LittleEndian.PutUint16(head[4:], uint16(8+2*len(names)))
	binary.LittleEndian.PutUint16(head[8:], 0)
	binary.LittleEndian.PutUint16(head[10:], uint16(2*(len(substitute)-1)))
	binary.LittleEndian.PutUint16(head[12:], uint16(2*len(substitute)))
	binary.LittleEndian.PutUint16(head[14:], uint16(2*(len(print)-1)))
	buf := append(head, unsafe.Slice((*byte)(unsafe.Pointer(&names[0])), 2*len(names))...)

	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	h, err := windows.CreateFile(name, windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(h) }()
	var n uint32
	return windows.DeviceIoControl(h, windows.FSCTL_SET_REPARSE_POINT, &buf[0], uint32(len(buf)), nil, 0, &n, nil)
}
