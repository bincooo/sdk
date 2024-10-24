package runtime

import (
	"fmt"
	"runtime"
)

func Goid() (id int64) {
	buf := make([]byte, 64)
	buf = buf[:runtime.Stack(buf, false)]
	_, err := fmt.Sscanf(string(buf), "goroutine %d [", &id)
	if err != nil {
		panic(err)
	}
	return
}
