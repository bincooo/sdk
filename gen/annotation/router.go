package annotation

import (
	"fmt"
	"go/ast"
)

type Router struct {
	Path   string `annotation:"name=path,default=/"`
	Method string `annotation:"name=method,default=GET"`
}

var _ M = (*Router)(nil)

func (r Router) Name() string {
	return "router"
}

func (r Router) Match(node ast.Node) (err error) {
	if fd, ok := node.(*ast.FuncDecl); ok && MethodReceiver(fd) == "" {
		return fmt.Errorf("`@Router` expected method receiver, but got empty for %s", fd.Name.Name)
	}
	return
}

func (r Router) As() (_ M) {
	return
}
