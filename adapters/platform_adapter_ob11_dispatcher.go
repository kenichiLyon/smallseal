package adapters

import (
	"context"
	"errors"
	"sync"
)

const (
	ob11DispatchWorkerCount = 1
	ob11DispatchQueueSize   = 256
)

type ob11DispatchJob struct {
	postType string
	payload  []byte
}

// ob11EventDispatcher limits frame handling concurrency to avoid unbounded goroutine growth.
type ob11EventDispatcher struct {
	ctx     context.Context
	queue   chan ob11DispatchJob
	handler func(ob11DispatchJob)
	wg      sync.WaitGroup
}

func newOB11EventDispatcher(ctx context.Context, workers, queueSize int, handler func(ob11DispatchJob)) *ob11EventDispatcher {
	if workers <= 0 {
		workers = 1
	}
	if queueSize <= 0 {
		queueSize = 1
	}

	d := &ob11EventDispatcher{
		ctx:     ctx,
		queue:   make(chan ob11DispatchJob, queueSize),
		handler: handler,
	}

	for i := 0; i < workers; i++ {
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			for {
				select {
				case <-d.ctx.Done():
					return
				case job := <-d.queue:
					d.handler(job)
				}
			}
		}()
	}

	return d
}

func (d *ob11EventDispatcher) submit(ctx context.Context, job ob11DispatchJob) error {
	if d == nil {
		return errors.New("ob11 dispatcher not initialized")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := d.ctx.Err(); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-d.ctx.Done():
		return d.ctx.Err()
	case d.queue <- job:
		return nil
	}
}

func (d *ob11EventDispatcher) wait() {
	if d == nil {
		return
	}
	d.wg.Wait()
}
