package runtime

import (
	"runtime"
	"strconv"
	"strings"
)

func Goid() (id int64) {
	buf := make([]byte, 64)
	buf = buf[:runtime.Stack(buf, false)]
	str := strings.Split(string(buf), " [running]:")[0]
	str = strings.TrimPrefix(str, "goroutine ")
	id, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		id = -1
	}
	return
}
