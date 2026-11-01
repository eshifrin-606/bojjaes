package main

import (
	"os"
	"syscall"
)

// Go applies these set*id calls to every thread of the process, not just the caller.
var osPrivileges = privileges{
	geteuid:   os.Geteuid,
	lchown:    os.Lchown,
	setgroups: syscall.Setgroups,
	setgid:    syscall.Setgid,
	setuid:    syscall.Setuid,
}
