package closer

import (
	"sync"
)

var (
	closers []func() error
	mu      sync.Mutex
)

func Add(closer func() error) {
	mu.Lock()
	defer mu.Unlock()
	closers = append(closers, closer)
}

func CloseAll() error {
	for _, closer := range closers {
		if err := closer(); err != nil {
			return err
		}
	}
	return nil
}

func Wait() {

}
