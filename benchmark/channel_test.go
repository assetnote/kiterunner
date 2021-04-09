package benchmark

import (
	"context"
	"sync"
	"testing"
)

func BenchmarkWaitGroup(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			wg.Done()
		}()
		wg.Wait()
	}
}

func BenchmarkChannel(b *testing.B) {
	for n := 0; n < b.N; n++ {
		done := make(chan bool)
		go func() {
			done <- true
		}()
		<-done
	}
}

func BenchmarkSelectChannel(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for n := 0; n < b.N; n++ {
		done := make(chan bool)
		go func() {
			select { case <-ctx.Done():
			case done <- true:
			}
		}()
		<-done
	}
}

func BenchmarkDoubleSelectChannel(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for n := 0; n < b.N; n++ {
		done := make(chan bool)
		go func() {
			select {
			case <-ctx.Done():
			case done <- true:
			}
		}()
		select {
		case <-ctx.Done():
		case <-done:
		}
	}
}
