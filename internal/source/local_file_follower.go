package source

import (
	"bytes"
	"context"
	"fmt"
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
	Lines              []byte
	OversizedLineCount int
	Reset              bool
	State              FollowState
	Err                error
}

type LocalFileFollower struct {
	path     string
	interval time.Duration
	events   chan FollowBatch
	started  chan struct{}
	done     chan struct{}

	file                *os.File
	info                os.FileInfo
	offset              int64
	pending             []byte
	discardingOversized bool
	initialized         bool

	closeEvents sync.Once
	startRun    sync.Once
}

func NewLocalFileFollower(path string, interval time.Duration) *LocalFileFollower {
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	return &LocalFileFollower{
		path: path, interval: interval,
		events:  make(chan FollowBatch, 16),
		started: make(chan struct{}),
		done:    make(chan struct{}),
	}
}

func (f *LocalFileFollower) Events() <-chan FollowBatch { return f.events }

// InitialRead opens the input once and retains the descriptor for Run. Only
// newline-terminated data is returned; a trailing fragment remains pending.
func (f *LocalFileFollower) InitialRead(ctx context.Context) (FollowBatch, error) {
	if f.initialized {
		return FollowBatch{}, fmt.Errorf("follower already initialized")
	}
	if err := ctx.Err(); err != nil {
		return FollowBatch{}, err
	}
	file, err := os.Open(f.path)
	if err != nil {
		return FollowBatch{}, fmt.Errorf("open followed file: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return FollowBatch{}, fmt.Errorf("stat followed file: %w", err)
	}
	f.file, f.info, f.initialized = file, info, true
	data, err := f.readAvailable()
	if err != nil {
		_ = file.Close()
		f.file = nil
		return FollowBatch{}, err
	}
	lines, oversized := f.completeLines(data)
	return FollowBatch{Lines: lines, OversizedLineCount: oversized, State: FollowLive}, nil
}

func (f *LocalFileFollower) Run(ctx context.Context) {
	f.startRun.Do(func() { close(f.started) })
	defer close(f.done)
	defer f.closeEvents.Do(func() { close(f.events) })
	defer func() {
		if f.file != nil {
			_ = f.file.Close()
			f.file = nil
		}
	}()

	if !f.initialized {
		batch, err := f.InitialRead(ctx)
		if err != nil {
			f.send(ctx, FollowBatch{State: FollowRetrying, Err: err})
		} else {
			f.send(ctx, batch)
		}
	}

	backoff := f.interval
	wasRetrying := false
	timer := time.NewTimer(f.interval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		retrying, err := f.poll(ctx)
		if err != nil {
			f.send(ctx, FollowBatch{State: FollowRetrying, Err: err})
			retrying = true
		}
		if retrying {
			if wasRetrying {
				backoff = min(backoff*2, 5*time.Second)
			} else {
				backoff = f.interval
			}
		} else {
			backoff = f.interval
			if wasRetrying {
				f.send(ctx, FollowBatch{State: FollowLive})
			}
		}
		wasRetrying = retrying
		timer.Reset(backoff)
	}
}

// Close waits for Run to release its file descriptor. The Run context must be
// cancelled before calling Close.
func (f *LocalFileFollower) Close() error {
	<-f.started
	<-f.done
	return nil
}

// Discard closes an initialized follower that has not been passed to Run.
func (f *LocalFileFollower) Discard() error {
	if f.file == nil {
		return nil
	}
	err := f.file.Close()
	f.file = nil
	f.initialized = false
	return err
}

func (f *LocalFileFollower) poll(ctx context.Context) (bool, error) {
	pathInfo, err := os.Stat(f.path)
	if err != nil {
		return true, fmt.Errorf("stat followed path: %w", err)
	}

	if f.file == nil {
		return f.openReplacement(ctx, pathInfo)
	}
	if !os.SameFile(f.info, pathInfo) {
		// Drain bytes written to the renamed file before switching. The reset
		// event follows this event, so the replacement remains the final state.
		data, readErr := f.readAvailable()
		if readErr != nil {
			return true, readErr
		}
		if lines, oversized := f.completeLines(data); len(lines) > 0 || oversized > 0 {
			f.send(ctx, FollowBatch{
				Lines: lines, OversizedLineCount: oversized, State: FollowLive,
			})
		}
		return f.openReplacement(ctx, pathInfo)
	}
	if pathInfo.Size() < f.offset {
		f.offset = 0
		f.pending = nil
		f.discardingOversized = false
		data, readErr := f.readAvailable()
		if readErr != nil {
			return true, readErr
		}
		lines, oversized := f.completeLines(data)
		f.send(ctx, FollowBatch{
			Lines: lines, OversizedLineCount: oversized, Reset: true, State: FollowLive,
		})
		return false, nil
	}

	data, err := f.readAvailable()
	if err != nil {
		return true, err
	}
	if lines, oversized := f.completeLines(data); len(lines) > 0 || oversized > 0 {
		f.send(ctx, FollowBatch{
			Lines: lines, OversizedLineCount: oversized, State: FollowLive,
		})
	}
	return false, nil
}

func (f *LocalFileFollower) openReplacement(ctx context.Context, info os.FileInfo) (bool, error) {
	replacement, err := os.Open(f.path)
	if err != nil {
		return true, fmt.Errorf("open replacement: %w", err)
	}
	if f.file != nil {
		_ = f.file.Close()
	}
	f.file, f.info, f.offset, f.pending = replacement, info, 0, nil
	f.discardingOversized = false
	data, err := f.readAvailable()
	if err != nil {
		_ = replacement.Close()
		f.file = nil
		return true, err
	}
	lines, oversized := f.completeLines(data)
	f.send(ctx, FollowBatch{
		Lines: lines, OversizedLineCount: oversized, Reset: true, State: FollowLive,
	})
	return false, nil
}

func (f *LocalFileFollower) readAvailable() ([]byte, error) {
	if f.file == nil {
		return nil, fmt.Errorf("followed file is not open")
	}
	if _, err := f.file.Seek(f.offset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek followed file: %w", err)
	}
	data, err := io.ReadAll(f.file)
	if err != nil {
		return nil, fmt.Errorf("read followed file: %w", err)
	}
	f.offset += int64(len(data))
	return data, nil
}

func (f *LocalFileFollower) completeLines(data []byte) ([]byte, int) {
	const maxLineSize = 10 * 1024 * 1024
	oversized := 0
	if f.discardingOversized {
		newline := bytes.IndexByte(data, '\n')
		if newline < 0 {
			return nil, 0
		}
		f.discardingOversized = false
		oversized++
		data = data[newline+1:]
	}
	all := append(append(f.pending[:0:0], f.pending...), data...)
	last := bytes.LastIndexByte(all, '\n')
	if last < 0 {
		if len(all) > maxLineSize {
			f.pending = nil
			f.discardingOversized = true
		} else {
			f.pending = all
		}
		return nil, oversized
	}
	complete := append([]byte(nil), all[:last+1]...)
	f.pending = append(f.pending[:0], all[last+1:]...)
	if len(f.pending) > maxLineSize {
		f.pending = nil
		f.discardingOversized = true
	}
	return complete, oversized
}

func (f *LocalFileFollower) send(ctx context.Context, event FollowBatch) {
	select {
	case f.events <- event:
	case <-ctx.Done():
	}
}
