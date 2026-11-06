package running

import (
	"strings"

	"golang.org/x/sys/unix"
)

func programs() []string {
	procs, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return nil
	}
	var out []string
	for _, p := range procs {
		// argc, then the path the process was started from
		args, err := unix.SysctlRaw("kern.procargs2", int(p.Proc.P_pid))
		if err != nil || len(args) < 4 {
			continue
		}
		if path, _, _ := strings.Cut(string(args[4:]), "\x00"); path != "" {
			out = append(out, path)
		}
	}
	return out
}
