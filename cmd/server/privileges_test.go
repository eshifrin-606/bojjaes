package main

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"
)

type privilegeCalls struct {
	calls  []string
	failAt string
}

var errStep = errors.New("step failed")

func (r *privilegeCalls) record(call string) error {
	r.calls = append(r.calls, call)
	if r.failAt != "" && strings.HasPrefix(call, r.failAt) {
		return errStep
	}
	return nil
}

func fakePrivileges(euid int, r *privilegeCalls) privileges {
	return privileges{
		geteuid: func() int { return euid },
		lchown: func(path string, uid, gid int) error {
			return r.record("lchown " + path + " " + strconv.Itoa(uid) + ":" + strconv.Itoa(gid))
		},
		setgroups: func(gids []int) error {
			return r.record("setgroups " + strconv.Itoa(len(gids)))
		},
		setgid: func(gid int) error {
			return r.record("setgid " + strconv.Itoa(gid))
		},
		setuid: func(uid int) error {
			if err := r.record("setuid " + strconv.Itoa(uid)); err != nil {
				return err
			}
			euid = uid
			return nil
		},
	}
}

func TestClaimVolumeAsRootChownsOnlyTheMountToNonroot(t *testing.T) {
	var r privilegeCalls

	if err := fakePrivileges(0, &r).claimVolumeAndDropRoot("/data"); err != nil {
		t.Fatalf("claimVolumeAndDropRoot: %v", err)
	}

	if !slices.Contains(r.calls, "lchown /data 65532:65532") {
		t.Errorf("calls = %q, want lchown /data 65532:65532", r.calls)
	}
}

func TestClaimVolumeIsANoOpWhenNotRoot(t *testing.T) {
	var r privilegeCalls

	if err := fakePrivileges(1000, &r).claimVolumeAndDropRoot("/data"); err != nil {
		t.Fatalf("claimVolumeAndDropRoot: %v", err)
	}

	if len(r.calls) != 0 {
		t.Errorf("calls = %q, want none", r.calls)
	}
}

func TestClaimVolumeDropsGroupsThenGidThenUid(t *testing.T) {
	var r privilegeCalls

	if err := fakePrivileges(0, &r).claimVolumeAndDropRoot("/data"); err != nil {
		t.Fatalf("claimVolumeAndDropRoot: %v", err)
	}

	want := []string{"lchown /data 65532:65532", "setgroups 0", "setgid 65532", "setuid 65532"}
	if !slices.Equal(r.calls, want) {
		t.Errorf("calls = %q, want %q", r.calls, want)
	}
}

func TestClaimVolumeWithNoVolumeStillDropsRoot(t *testing.T) {
	var r privilegeCalls

	if err := fakePrivileges(0, &r).claimVolumeAndDropRoot(""); err != nil {
		t.Fatalf("claimVolumeAndDropRoot: %v", err)
	}

	want := []string{"setgroups 0", "setgid 65532", "setuid 65532"}
	if !slices.Equal(r.calls, want) {
		t.Errorf("calls = %q, want %q", r.calls, want)
	}
}

func TestClaimVolumeStopsAtTheFirstFailingStep(t *testing.T) {
	steps := []string{"lchown", "setgroups", "setgid", "setuid"}
	for i, step := range steps {
		t.Run(step, func(t *testing.T) {
			r := privilegeCalls{failAt: step}

			err := fakePrivileges(0, &r).claimVolumeAndDropRoot("/data")

			if !errors.Is(err, errStep) {
				t.Errorf("err = %v, want %v", err, errStep)
			}
			if len(r.calls) != i+1 {
				t.Errorf("calls = %q, want none after %s", r.calls, step)
			}
		})
	}
}

func TestClaimVolumeFailsWhenStillRootAfterTheDrop(t *testing.T) {
	var r privilegeCalls
	p := fakePrivileges(0, &r)
	p.setuid = func(int) error { return nil }

	if err := p.claimVolumeAndDropRoot("/data"); err == nil {
		t.Error("claimVolumeAndDropRoot = nil, want an error while euid is still 0")
	}
}
