//go:build !linux

package main

import (
	"errors"
	"os"
)

// Only the Linux image starts as root; elsewhere euid is never 0 in practice,
// and if it is, refusing beats running as root.
var osPrivileges = privileges{
	geteuid:   os.Geteuid,
	lchown:    func(string, int, int) error { return errors.ErrUnsupported },
	setgroups: func([]int) error { return errors.ErrUnsupported },
	setgid:    func(int) error { return errors.ErrUnsupported },
	setuid:    func(int) error { return errors.ErrUnsupported },
}
