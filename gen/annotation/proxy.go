package annotation

import (
	"fmt"
	"go/ast"
)

type Proxy struct {
	Target string `annotation:"name=target,default="`
}

var _ M = (*Proxy)(nil)

func (p Proxy) Name() string {
	return "proxy"
}

func (p Proxy) Match(node ast.Node) (err error) {
	if p.Target == "" {
		return fmt.Errorf("please specify the proxy function")
	}

	if _, ok := node.(*ast.TypeSpec); !ok {
		err = fmt.Errorf("the position of the `@proxy` annotation is incorrect")
	}
	return
}

func (p Proxy) As() (_ M) {
	return
}

func MethodReceiver(decl *ast.FuncDecl) string {
	if decl.Recv == nil {
		return ""
	}

	for _, v := range decl.Recv.List {
		switch rv := v.Type.(type) {
		case *ast.Ident:
			return rv.Name
		case *ast.StarExpr:
			return rv.X.(*ast.Ident).Name
		case *ast.UnaryExpr:
			return rv.X.(*ast.Ident).Name
		}
	}
	return ""
}
