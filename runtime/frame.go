package runtime

import (
	"runtime"
)

func CallerFrame(matched func(frame runtime.Frame) bool) *runtime.Frame {
	pcs := make([]uintptr, 10)
	depth := runtime.Callers(1, pcs)
	frames := runtime.CallersFrames(pcs[:depth])
	for f, next := frames.Next(); next; f, next = frames.Next() {
		if matched(f) {
			return &f
		}
	}
	return nil
}
