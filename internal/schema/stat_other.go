//go:build !linux

package schema

import "os"

// Statx reads path's [Stat] through os.Lstat. Off Linux there is no statx, so the created time
// is always zero. No symbolic link is followed.
func Statx(path string) (Stat, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Stat{}, err
	}
	return Stat{Size: info.Size(), Mode: info.Mode(), Modified: info.ModTime()}, nil
}
