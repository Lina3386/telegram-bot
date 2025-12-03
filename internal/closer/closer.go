package closer

import (
	"io"
	"sync"
)

var (
	closers []io.Closer
	mu      sync.Mutex
)

func Add(closer io.Closer) {
	mu.Lock()
	defer mu.Unlock()
	closers = append(closers, closer)
}

func CloseAll() {
	mu.Lock()
	defer mu.Unlock()

	for _, closer := range closers {
		_ = closer.Close()
	}
}

func Wait() {
	
}


