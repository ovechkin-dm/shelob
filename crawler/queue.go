package crawler

type Queue[T any] interface {
	Put(item T)
	Get() T
	Out() <-chan T
	Close()
}

type UnboundedQueue[T any] struct {
	buf  []T
	pos  int
	in   chan T
	done chan struct{}
	out  chan T
}

func (q *UnboundedQueue[T]) Put(item T) {
	q.in <- item
}

func (q *UnboundedQueue[T]) Get() T {
	return <-q.Out()
}

func (q *UnboundedQueue[T]) Out() <-chan T {
	return q.out
}

func (q *UnboundedQueue[T]) Close() {
	q.done <- struct{}{}
}

func (q *UnboundedQueue[T]) start() {
	for {
		select {
		case item := <-q.in:
			q.buf = append(q.buf, item)
			for q.pos < len(q.buf) {
				toSend := q.buf[q.pos]
				select {
				case q.out <- toSend:
					q.pos++
				case it := <-q.in:
					q.buf = append(q.buf, it)
				case <-q.done:
					close(q.out)
				}
			}
		case <-q.done:
			close(q.out)
			return
		}
		q.buf = q.buf[q.pos:]
		q.pos = 0
	}
}

func NewUnboundedQueue[T any]() *UnboundedQueue[T] {
	q := &UnboundedQueue[T]{
		in:   make(chan T),
		done: make(chan struct{}),
		out:  make(chan T),
	}
	go q.start()
	return q
}
