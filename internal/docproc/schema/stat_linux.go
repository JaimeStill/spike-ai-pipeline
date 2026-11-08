package schema

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// Statx reads path's [Stat] through os.Lstat, and its created time through statx, which reports
// a birth time only where the filesystem records one: when statx's mask lacks STATX_BTIME, the
// created time is zero. No symbolic link is followed.
func Statx(path string) (Stat, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Stat{}, err
	}
	st := Stat{Size: info.Size(), Mode: info.Mode(), Modified: info.ModTime()}
	var stx unix.Statx_t
	if err := unix.Statx(unix.AT_FDCWD, path, unix.AT_SYMLINK_NOFOLLOW, unix.STATX_BTIME, &stx); err != nil {
		return Stat{}, &os.PathError{Op: "statx", Path: path, Err: err}
	}
	if stx.Mask&unix.STATX_BTIME != 0 {
		st.Created = time.Unix(stx.Btime.Sec, int64(stx.Btime.Nsec))
	}
	return st, nil
}
