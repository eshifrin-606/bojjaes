package main

import (
	"errors"
	"fmt"
)

// The uid/gid of the distroless nonroot user.
const nonroot = 65532

type privileges struct {
	geteuid   func() int
	lchown    func(path string, uid, gid int) error
	setgroups func(gids []int) error
	setgid    func(gid int) error
	setuid    func(uid int) error
}

func (p privileges) claimVolumeAndDropRoot(volume string) error {
	if p.geteuid() != 0 {
		return nil
	}
	if volume != "" {
		// Lchown, and only the mount itself: a symlink planted on the volume must not redirect it.
		if err := p.lchown(volume, nonroot, nonroot); err != nil {
			return fmt.Errorf("chown %s: %w", volume, err)
		}
	}
	if err := p.setgroups(nil); err != nil {
		return fmt.Errorf("setgroups: %w", err)
	}
	// Gid first: once uid is nonroot, the process may no longer change its gid.
	if err := p.setgid(nonroot); err != nil {
		return fmt.Errorf("setgid: %w", err)
	}
	if err := p.setuid(nonroot); err != nil {
		return fmt.Errorf("setuid: %w", err)
	}
	if p.geteuid() == 0 {
		return errors.New("still root after dropping privileges")
	}
	return nil
}

func claimVolumeAndDropRoot(volume string) error {
	return osPrivileges.claimVolumeAndDropRoot(volume)
}
