package storage

import (
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWithLockSerializesConcurrentCallers(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "project.lock")
	var active atomic.Int32
	var overlap atomic.Bool
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := WithLock(lockPath, time.Second, func() error {
				if active.Add(1) != 1 {
					overlap.Store(true)
				}
				time.Sleep(20 * time.Millisecond)
				active.Add(-1)
				return nil
			})
			if err != nil {
				t.Errorf("WithLock() error = %v", err)
			}
		}()
	}
	wg.Wait()
	if overlap.Load() {
		t.Fatal("WithLock() allowed overlapping critical sections")
	}
}

func TestWithLockTimesOut(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "project.lock")
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- WithLock(lockPath, time.Second, func() error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started
	err := WithLock(lockPath, 30*time.Millisecond, func() error { return nil })
	close(release)
	if lockErr := <-done; lockErr != nil {
		t.Fatal(lockErr)
	}
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("WithLock() error = %v", err)
	}
}
