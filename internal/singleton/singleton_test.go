//go:build linux

package singleton

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

func TestAcquireTwice(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	ok, err := Acquire("test")
	if err != nil || !ok {
		t.Fatalf("первая блокировка: %v, %v", ok, err)
	}
	// The garbage collector must not release the lock.
	runtime.GC()
	runtime.GC()
	ok, err = Acquire("test")
	if err != nil || ok {
		t.Fatalf("вторая блокировка должна отказать: %v, %v", ok, err)
	}
}

// A child process sees the parent's lock, like a second launch of the program.
func TestAcquireAcrossProcesses(t *testing.T) {
	if os.Getenv("SINGLETON_CHILD") == "1" {
		ok, _ := Acquire("test-proc")
		if ok {
			os.Exit(0)
		}
		os.Exit(3)
	}
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	if ok, err := Acquire("test-proc"); err != nil || !ok {
		t.Fatalf("блокировка родителя: %v, %v", ok, err)
	}
	runtime.GC()

	cmd := exec.Command(os.Args[0], "-test.run=TestAcquireAcrossProcesses")
	cmd.Env = append(os.Environ(), "SINGLETON_CHILD=1")
	err := cmd.Run()
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 3 {
		t.Fatalf("дочерний процесс должен получить отказ (код 3), получили %v", err)
	}
}
