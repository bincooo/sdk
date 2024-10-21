package runtime

import (
	"sync"
	"testing"
)

func TestThreadLocal(t *testing.T) {
	threadLocal := NewThreadLocal(func() []string {
		return make([]string, 0)
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer threadLocal.Remove()

		slice := []string{"hello", "world"}
		threadLocal.Set(slice)
		slice[1] = "nil"
		t.Log(threadLocal.Get())
	}()

	wg.Wait()
	t.Log(threadLocal.Get())
	t.Log("=======")
}
