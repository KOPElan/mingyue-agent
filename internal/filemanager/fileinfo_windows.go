//go:build windows

package filemanager

import (
	"os"
)

func getOwnerAndGroup(info os.FileInfo) (owner uint32, group uint32, ok bool) {
	// Windows doesn't have Unix-style UID/GID
	return 0, 0, false
}
