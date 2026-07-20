package source

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"time"
)

type FollowState string

const (
	FollowLive     FollowState = "live"
	FollowRetrying FollowState = "retrying"
)

type FollowBatch struct {
	Lines []byte
	Reset bool
	State FollowState
	Err   error
}

type LocalFileFollower struct {
	path          string
	interval      time.Duration
	events        chan FollowBatch
	close         sync.Once
	initialOffset *int64
}

func NewLocalFileFollowerAtOffset(path string, interval time.Duration, offset int64) *LocalFileFollower {
	follower := NewLocalFileFollower(path, interval)
	follower.initialOffset = &offset
	return follower
}

func NewLocalFileFollower(path string, interval time.Duration) *LocalFileFollower {
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	return &LocalFileFollower{path: path, interval: interval, events: make(chan FollowBatch, 16)}
}

func (f *LocalFileFollower) Events() <-chan FollowBatch { return f.events }

func (f *LocalFileFollower) Run(ctx context.Context) {
	defer f.close.Do(func() { close(f.events) })
	var file *os.File
	var info os.FileInfo
	var offset int64
	var pending []byte
	backoff := f.interval
	resetOnOpen := false

	openFile := func(atEnd bool) error {
		opened, err := os.Open(f.path)
		if err != nil {
			return err
		}
		stat, err := opened.Stat()
		if err != nil {
			opened.Close()
			return err
		}
		start := int64(0)
		if atEnd {
			start = stat.Size()
		}
		if f.initialOffset != nil {
			start = min(*f.initialOffset, stat.Size())
			f.initialOffset = nil
		}
		file, info, offset = opened, stat, start
		pending = nil
		return nil
	}
	if err := openFile(true); err != nil {
		f.send(ctx, FollowBatch{State: FollowRetrying, Err: err})
	} else {
		defer func() {
			if file != nil {
				_ = file.Close()
			}
		}()
		f.send(ctx, FollowBatch{State: FollowLive})
	}

	timer := time.NewTimer(backoff)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		if file == nil {
			if err := openFile(!resetOnOpen); err != nil {
				f.send(ctx, FollowBatch{State: FollowRetrying, Err: err})
				backoff = min(backoff*2, 5*time.Second)
				timer.Reset(backoff)
				continue
			}
			if resetOnOpen {
				data, readErr := readFrom(file, 0)
				if readErr != nil {
					f.send(ctx, FollowBatch{State: FollowRetrying, Err: readErr})
				} else {
					offset = int64(len(data))
					f.emitComplete(ctx, &pending, data, true)
				}
				resetOnOpen = false
			} else {
				f.send(ctx, FollowBatch{State: FollowLive})
			}
			backoff = f.interval
			timer.Reset(backoff)
			continue
		}

		pathInfo, err := os.Stat(f.path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				_ = file.Close()
				file = nil
				resetOnOpen = true
			}
			f.send(ctx, FollowBatch{State: FollowRetrying, Err: err})
			timer.Reset(backoff)
			continue
		}

		reset := false
		if !os.SameFile(info, pathInfo) {
			if data, readErr := readFrom(file, offset); readErr == nil {
				f.emitComplete(ctx, &pending, data, false)
			}
			_ = file.Close()
			file, err = os.Open(f.path)
			if err != nil {
				file = nil
				f.send(ctx, FollowBatch{State: FollowRetrying, Err: err})
				timer.Reset(backoff)
				continue
			}
			info = pathInfo
			offset = 0
			pending = nil
			reset = true
		} else if pathInfo.Size() < offset {
			offset = 0
			pending = nil
			reset = true
		}

		data, err := readFrom(file, offset)
		if err != nil {
			f.send(ctx, FollowBatch{State: FollowRetrying, Err: err})
			timer.Reset(backoff)
			continue
		}
		offset += int64(len(data))
		f.emitComplete(ctx, &pending, data, reset)
		timer.Reset(f.interval)
	}
}

func readFrom(file *os.File, offset int64) ([]byte, error) {
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	return io.ReadAll(file)
}

func (f *LocalFileFollower) emitComplete(ctx context.Context, pending *[]byte, data []byte, reset bool) {
	all := append(append((*pending)[:0:0], *pending...), data...)
	last := bytes.LastIndexByte(all, '\n')
	if last < 0 {
		*pending = all
		if reset {
			f.send(ctx, FollowBatch{Reset: true, State: FollowLive})
		}
		return
	}
	complete := append([]byte(nil), all[:last+1]...)
	*pending = append((*pending)[:0], all[last+1:]...)
	f.send(ctx, FollowBatch{Lines: complete, Reset: reset, State: FollowLive})
}

func (f *LocalFileFollower) send(ctx context.Context, event FollowBatch) {
	select {
	case f.events <- event:
	case <-ctx.Done():
	}
}
