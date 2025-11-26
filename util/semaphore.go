package util

type Semaphore struct {
	ch chan struct{}
}

func NewSemaphore(maxSignals int) *Semaphore {
	return &Semaphore{
		ch: make(chan struct{}, maxSignals),
	}
}

func (s *Semaphore) Acquire() {
	s.ch <- struct{}{}
}

func (s *Semaphore) Release() {
	<-s.ch
}
