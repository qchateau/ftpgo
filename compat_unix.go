// +build linux darwin

package main

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
)

func FileInfoToUserGroupNames(info os.FileInfo) (string, string) {
	userName := "unknown"
	groupName := "unknown"

	if sys, ok := info.Sys().(*syscall.Stat_t); ok {
		if user, err := user.LookupId(strconv.FormatUint(uint64(sys.Uid), 10)); err == nil {
			userName = user.Username
		}
		if grp, err := user.LookupGroupId(strconv.FormatUint(uint64(sys.Gid), 10)); err == nil {
			groupName = grp.Name
		}
	}
	return userName, groupName
}
