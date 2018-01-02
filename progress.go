// Package progress provides io.Reader and io.Writer with progress and remaining time estimation.
//  ctx := context.Background()
//
//  // get a reader and the total expected number of bytes
//  s := `Now that's what I call progress`
//  size := len(s)
//  r := progress.NewReader(strings.NewReader(s))
//
//  // Start a goroutine printing progress
//  go func(){
//  	defer log.Printf("done")
//  	interval := 1 * time.Second
//  	progressChan := progress.NewTicker(ctx, r, size, interval)
//  	for {
//  		select {
//  		case progress, ok := <-progressChan:
//  			if !ok {
//  				// if ok is false, the process is finished
//  				return
//  			}
//  			log.Printf("about %v remaining...", progress.Remaining())
//  		}
//  	}
//  }()
//
//  // use the Reader as normal
//  if _, err := io.Copy(dest, r); err != nil {
//  	log.Fatalln(err)
//  }
package progress

import (
	"context"
	"time"
)

// Counter counts bytes.
// Both Reader and Writer are Counter types.
type Counter interface {
	// N gets the current count value.
	// For readers and writers, this is the number of bytes
	// read or written.
	// For other contexts, the number may be anything.
	N() int64
}

// Progress represents a moment of progress.
type Progress struct {
	n         float64
	size      float64
	estimated time.Time
}

// N gets the total number of bytes read or written
// so far.
func (p Progress) N() int64 {
	return int64(p.n)
}

// Size gets the total number of bytes that are expected to
// be read or written.
func (p Progress) Size() int64 {
	return int64(p.size)
}

// Started gets whether the operation has started or not.
func (p Progress) Started() bool {
	return p.n > 0
}

// Complete gets whether the operation is complete or not.
func (p Progress) Complete() bool {
	return p.n >= p.size
}

// Percent calculates the percentage complete.
func (p Progress) Percent() float64 {
	if p.n == 0 {
		return 0
	}
	if p.n == p.size {
		return 100
	}
	return 100.0 / (p.size / p.n)
}

// Remaining gets the amount of time until the operation is
// expected to be finished. Use Estimated to get a fixed completion time.
func (p Progress) Remaining() time.Duration {
	return p.estimated.Sub(time.Now())
}

// Estimated gets the time at which the operation is expected
// to finish. Use Reamining to get a Duration.
func (p Progress) Estimated() time.Time {
	return p.estimated
}

// NewTicker gets a channel on which ticks of Progress are sent
// at duration d intervals until the operation is complete at which point
// the channel is closed.
// The counter is either a Reader or Writer (or any type that can report its progress).
// The size is the total number of expected bytes being read or written.
// If the context cancels the operation, the channel is closed.
func NewTicker(ctx context.Context, counter Counter, size int64, d time.Duration) <-chan Progress {
	var (
		started time.Time
		ch      = make(chan Progress)
	)
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				// context has finished - exit
				return
			case <-time.After(d):
				progress := Progress{
					n:    float64(counter.N()),
					size: float64(size),
				}
				if started.IsZero() {
					if progress.Started() {
						started = time.Now()
					}
				} else {
					now := time.Now()
					ratio := progress.n / progress.size
					past := float64(now.Sub(started))
					future := time.Duration(past / ratio)
					progress.estimated = started.Add(future)
				}
				ch <- progress
				if progress.Complete() {
					return
				}
			}
		}
	}()
	return ch
}
