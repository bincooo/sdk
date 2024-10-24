package gen

import (
	annotation "github.com/YReshetko/go-annotation/pkg"
	gen "github.com/bincooo/sdk/gen/annotation"
	"github.com/bincooo/sdk/gen/internal/core"
)

func Alias[T gen.M]() {
	core.Alias[T]()
}

func Process() {
	annotation.Process()
}
