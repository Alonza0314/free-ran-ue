package util

type semaphore struct {
	ch chan struct{}
}

func NewSemaphore(maxSignals int) *semaphore {
	return &semaphore{
		ch: make(chan struct{}, maxSignals),
	}
}

func (s *semaphore) Acquire() {
	s.ch <- struct{}{}
}

func (s *semaphore) Release() {
	<-s.ch
}
