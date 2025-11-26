package util_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Alonza0314/free-ran-ue/util"
)

func TestSemaphoreBasic(t *testing.T) {
	sem := util.NewSemaphore(1)

	sem.Acquire()
	sem.Release()

	sem.Acquire()
	sem.Release()
}

func TestSemaphoreCapacity(t *testing.T) {
	maxSignals := 3
	sem := util.NewSemaphore(maxSignals)

	for i := 0; i < maxSignals; i++ {
		sem.Acquire()
	}

	for i := 0; i < maxSignals; i++ {
		sem.Release()
	}
}

func TestSemaphoreBlocking(t *testing.T) {
	sem := util.NewSemaphore(1)

	sem.Acquire()

	acquired := make(chan bool, 1)

	go func() {
		sem.Acquire()
		acquired <- true
	}()

	time.Sleep(100 * time.Millisecond)

	select {
	case <-acquired:
		t.Error("Acquire should be blocked, but it is not blocked")
	default:
	}

	sem.Release()

	select {
	case <-acquired:
	case <-time.After(1 * time.Second):
		t.Error("Acquire should be able to get after Release, but it timed out")
	}

	sem.Release()
}

func TestSemaphoreConcurrent(t *testing.T) {
	maxConcurrent := 5
	sem := util.NewSemaphore(maxConcurrent)

	var currentConcurrent int32
	var maxReached int32
	var wg sync.WaitGroup

	numGoroutines := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			sem.Acquire()

			current := atomic.AddInt32(&currentConcurrent, 1)

			for {
				maxVal := atomic.LoadInt32(&maxReached)
				if current <= maxVal {
					break
				}
				if atomic.CompareAndSwapInt32(&maxReached, maxVal, current) {
					break
				}
			}

			time.Sleep(10 * time.Millisecond)

			atomic.AddInt32(&currentConcurrent, -1)

			sem.Release()
		}()
	}

	wg.Wait()

	if maxReached > int32(maxConcurrent) {
		t.Errorf("concurrent number exceeds limit: expected at most %d, actual reached %d", maxConcurrent, maxReached)
	}

	if maxReached < int32(maxConcurrent) {
		t.Logf("warning: maximum concurrent number %d is less than limit %d, possibly because of goroutine scheduling problem", maxReached, maxConcurrent)
	}
}

func TestSemaphoreMultipleAcquireRelease(t *testing.T) {
	sem := util.NewSemaphore(2)

	iterations := 100
	var wg sync.WaitGroup

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem.Acquire()
			time.Sleep(1 * time.Millisecond)
			sem.Release()
		}()
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Error("test timed out, possibly because of deadlock")
	}
}

func TestSemaphoreZeroCapacity(t *testing.T) {
	sem := util.NewSemaphore(0)

	acquired := make(chan bool)

	go func() {
		sem.Acquire()
		acquired <- true
	}()

	select {
	case <-acquired:
		t.Error("semaphore with zero capacity should not be able to acquire")
	case <-time.After(200 * time.Millisecond):
	}
}
