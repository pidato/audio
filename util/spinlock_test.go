package util

import (
	"github.com/bytedance/gopkg/util/gopool"
	"sync"
	"testing"
)

func BenchmarkSpinLock_Lock(b *testing.B) {
	b.Run("Spinlock 1 thread", func(b *testing.B) {
		var l = &SpinLock{}
		for i := 0; i < b.N; i++ {
			l.Lock()
			l.Unlock()
		}
	})

	b.Run("Spinlock 2 threads", func(b *testing.B) {
		var l = &SpinLock{}
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < b.N; i++ {
					l.Lock()
					l.Unlock()
				}
			}()
		}
		wg.Wait()
	})
	b.Run("Spinlock 4 threads", func(b *testing.B) {
		var l = &SpinLock{}
		var wg sync.WaitGroup
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < b.N; i++ {
					l.Lock()
					l.Unlock()
				}
			}()
		}
		wg.Wait()
	})
	b.Run("Spinlock 8 threads", func(b *testing.B) {
		var l = &SpinLock{}
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < b.N; i++ {
					l.Lock()
					l.Unlock()
				}
			}()
		}
		wg.Wait()
	})

	b.Run("sync.Mutex", func(b *testing.B) {
		var l sync.Mutex
		for i := 0; i < b.N; i++ {
			l.Lock()
			l.Unlock()
		}
	})
	b.Run("sync.Mutex 2 threads", func(b *testing.B) {
		var l sync.Mutex
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < b.N; i++ {
					l.Lock()
					l.Unlock()
				}
			}()
		}
		wg.Wait()
	})
	b.Run("sync.Mutex 4 threads", func(b *testing.B) {
		var l sync.Mutex
		var wg sync.WaitGroup
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < b.N; i++ {
					l.Lock()
					l.Unlock()
				}
			}()
		}
		wg.Wait()
	})
	b.Run("sync.Mutex 8 threads", func(b *testing.B) {
		var l sync.Mutex
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < b.N; i++ {
					l.Lock()
					l.Unlock()
				}
			}()
		}
		wg.Wait()
	})
}

func BenchmarkGoPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		gopool.Go(func() {

		})
	}
}

type worker struct{}

func (w *worker) run() {
	for {

	}
}
