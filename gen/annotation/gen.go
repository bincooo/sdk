package annotation

import (
	"fmt"
	"go/ast"
)

type Gen struct {
	Target string `annotation:"name=target,default="`
}

var _ M = (*Gen)(nil)

func (Gen) Name() string {
	return "gen"
}

func (Gen) Match(node ast.Node) (err error) {
	if fd, ok := node.(*ast.FuncDecl); !ok || MethodReceiver(fd) != "" || fd.Name.Name != "main" {
		err = fmt.Errorf("the position of the `@Gen` annotation is incorrect, needed is main function (ast.FuncDecl)")
	}
	return
}

func (g Gen) As() (_ M) {
	return
}
