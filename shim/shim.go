// Package shim stamps and removes wedge shims: small executables on PATH
// that run a real binary where it is unpacked. Running it at its real path is
// the point — a tool that finds its own files from where it sits, as go finds
// its GOROOT, would not from a copy or a hard link.
package shim

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ExeName is name as the host spells an executable.
func ExeName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

// Stamper is where one tool keeps its shims and what they may point at.
type Stamper struct {
	// Tool names the tool in messages.
	Tool string
	// BinDir is where shims are stamped: the directory on the user's PATH.
	BinDir string
	// MalletDir holds wedge's mallet stamper and its templates.
	MalletDir string
	// Owned is the directory every binary the tool shims lives under. A shim
	// that points into it is one the tool stamped.
	Owned string
}

// Path is where the shim whose file is called file lives. A shim is named as
// it sits on disk, host suffix and all: ExeName adds one to a bare name.
func (s Stamper) Path(file string) string {
	return filepath.Join(s.BinDir, file)
}

// IsShim reports whether path is a shim this tool stamped: a wedge shim
// pointing into Owned. It is the guard for every write and delete: the bin
// directory may hold binaries the tool knows nothing about, and those are not
// its to replace or remove.
func (s Stamper) IsShim(path string) bool {
	target, ok := StampedTarget(path)
	if !ok {
		return false
	}
	rel, err := filepath.Rel(s.Owned, target)
	return err == nil && !filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// StampedTarget is the path a wedge shim runs. Its trailer is at the end of the file, or, in a macOS shim mallet
// signed, right before the code signature, which covers it
func StampedTarget(path string) (target string, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return "", false
	}
	if target, ok := trailerEndingAt(f, info.Size()); ok {
		return target, true
	}
	if end, ok := signatureOffset(f); ok {
		return trailerEndingAt(f, end)
	}
	return "", false
}

func trailerEndingAt(f *os.File, end int64) (string, bool) {
	if end < 8 {
		return "", false
	}
	tail := make([]byte, 8)
	if _, err := f.ReadAt(tail, end-8); err != nil {
		return "", false
	}
	magic := string(tail[4:])
	if magic != "SHIM" && magic != "SHM2" {
		return "", false
	}
	n := int64(binary.LittleEndian.Uint32(tail[:4]))
	if n == 0 || n+8 > end {
		return "", false
	}
	payload := make([]byte, n)
	if _, err := f.ReadAt(payload, end-8-n); err != nil {
		return "", false
	}
	if magic == "SHM2" {
		payload, _, _ = bytes.Cut(payload, []byte{0})
	}
	return string(payload), true
}

const (
	machoMagic64    = 0xfeedfacf
	lcCodeSignature = 0x1d
)

// signatureOffset is where a 64-bit Mach-O's code signature starts, read from its load commands
func signatureOffset(f *os.File) (int64, bool) {
	head := make([]byte, 32)
	if _, err := f.ReadAt(head, 0); err != nil || binary.LittleEndian.Uint32(head) != machoMagic64 {
		return 0, false
	}
	ncmds := binary.LittleEndian.Uint32(head[16:])
	cmds := make([]byte, binary.LittleEndian.Uint32(head[20:]))
	if _, err := f.ReadAt(cmds, 32); err != nil {
		return 0, false
	}
	for at, i := 0, uint32(0); i < ncmds && at+16 <= len(cmds); i++ {
		cmd, size := binary.LittleEndian.Uint32(cmds[at:]), int(binary.LittleEndian.Uint32(cmds[at+4:]))
		if cmd == lcCodeSignature {
			return int64(binary.LittleEndian.Uint32(cmds[at+8:])), true
		}
		if size < 8 {
			return 0, false
		}
		at += size
	}
	return 0, false
}

// Create stamps the shim file that runs source. mallet picks the wedge
// template itself (console or GUI, on Windows).
func (s Stamper) Create(file, source string, force bool) error {
	target := s.Path(file)

	// checked before anything else: whether the tool may write here does not
	// depend on its own tooling being installed
	if _, err := os.Stat(target); err == nil && !s.IsShim(target) && !force {
		return fmt.Errorf("%s exists and is not a %s shim — pass --force to replace it", target, s.Tool)
	}

	malletPath := filepath.Join(s.MalletDir, ExeName("mallet"))
	if _, err := os.Stat(malletPath); err != nil {
		return fmt.Errorf("mallet not found at %s", malletPath)
	}
	if s.stampedFromTemplate(target, source) {
		return nil
	}
	if err := os.MkdirAll(s.BinDir, 0755); err != nil {
		return err
	}

	out, err := exec.Command(malletPath, "--source-path", source, "--target-path", target).CombinedOutput() //nolint:gosec
	if err != nil {
		return fmt.Errorf("mallet: %w: %s", err, bytes.TrimSpace(out))
	}
	return nil
}

// stampedFromTemplate reports whether the shim at path is what mallet would
// stamp now: source, no arguments, on a template beside mallet.
func (s Stamper) stampedFromTemplate(path, source string) bool {
	b, err := os.ReadFile(path)
	if err != nil || len(b) < 8 || string(b[len(b)-4:]) != "SHIM" {
		return false
	}
	n := int(binary.LittleEndian.Uint32(b[len(b)-8 : len(b)-4]))
	if n+8 > len(b) || string(b[len(b)-8-n:len(b)-8]) != source {
		return false
	}
	body := b[:len(b)-8-n]
	for _, name := range []string{"wedge", "wedge-gui"} {
		if t, err := os.ReadFile(filepath.Join(s.MalletDir, ExeName(name))); err == nil && bytes.Equal(t, body) {
			return true
		}
	}
	return false
}

// Remove deletes the shim file. A file of that name that the tool did not
// stamp is left alone: removing its own package should never take somebody
// else's binary with it.
func (s Stamper) Remove(file string) error {
	target := s.Path(file)
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return nil
	}
	if !s.IsShim(target) {
		return nil
	}
	err := os.Remove(target)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
