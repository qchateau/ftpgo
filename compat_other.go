// +build !linux,!darwin

package main

import (
	"os"
)

func FileInfoToUserGroupNames(info os.FileInfo) (string, string) {
	userName := "unknown"
	groupName := "unknown"
	return userName, groupName
}
