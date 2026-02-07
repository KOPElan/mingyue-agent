//go:build linux || darwin

package filemanager

import (
	"os"
	"syscall"
)

func getOwnerAndGroup(info os.FileInfo) (owner uint32, group uint32, ok bool) {
	if stat, statOk := info.Sys().(*syscall.Stat_t); statOk {
		return stat.Uid, stat.Gid, true
	}
	return 0, 0, false
}
