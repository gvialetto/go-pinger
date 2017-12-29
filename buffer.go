package pinger

import (
	"container/list"
	"time"
)

type bufferPool struct {
	toRelease chan []byte
	toGive    chan []byte
}

type queuedBuffer struct {
	buf      []byte
	queuedAt time.Time
}

func newBufPool(bufSize int) *bufferPool {
	toRelease := make(chan []byte)
	toGive := make(chan []byte)

	go manageBufferPool(toRelease, toGive, bufSize, 30*time.Second)

	return &bufferPool{
		toRelease: toRelease,
		toGive:    toGive,
	}
}

func (bp *bufferPool) Get() []byte {
	return <-bp.toGive
}

func (bp *bufferPool) Release(buf []byte) {
	bp.toRelease <- buf
}

func manageBufferPool(
	toRelease chan []byte,
	toGive chan []byte,
	bufSize int,
	expirationTime time.Duration,
) {
	freeBuffers := list.New()
	for {
		// if we don't have any free buffers, add one
		if freeBuffers.Len() == 0 {
			freeBuffers.PushFront(queuedBuffer{
				queuedAt: time.Now(),
				buf:      make([]byte, bufSize),
			})
		}

		freeBuf := freeBuffers.Front()

		// TODO: this should probably be configurable
		select {
		case buf := <-toRelease:
			freeBuffers.PushFront(queuedBuffer{queuedAt: time.Now(), buf: buf})
		case toGive <- freeBuf.Value.(queuedBuffer).buf:
			freeBuffers.Remove(freeBuf)
		case <-time.After(expirationTime):
			el := freeBuffers.Front()
			for el != nil {
				if time.Since(el.Value.(queuedBuffer).queuedAt) > expirationTime {
					freeBuffers.Remove(el)
					el.Value = nil
				}
				el = freeBuffers.Front()
			}
		}
	}
}
