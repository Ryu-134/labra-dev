package queue

import (
	"context"
	"errors"
	"sync"
)

var ErrEmpty = errors.New("queue is empty")

type Job struct {
	Type    string
	Payload map[string]any
}

type Enqueuer interface {
	Enqueue(ctx context.Context, job Job) error
}

type Dequeuer interface {
	Dequeue(ctx context.Context) (Job, error)
}

type InMemoryQueue struct {
	mu   sync.Mutex
	jobs []Job
}

func NewInMemoryQueue() *InMemoryQueue {
	return &InMemoryQueue{
		jobs: make([]Job, 0),
	}
}

func (q *InMemoryQueue) Enqueue(_ context.Context, job Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.jobs = append(q.jobs, job)
	return nil
}

func (q *InMemoryQueue) Dequeue(_ context.Context) (Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.jobs) == 0 {
		return Job{}, ErrEmpty
	}

	job := q.jobs[0]
	q.jobs = q.jobs[1:]
	return job, nil
}

func (q *InMemoryQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.jobs)
}
