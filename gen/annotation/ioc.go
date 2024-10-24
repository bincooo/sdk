package annotation

import (
	"fmt"
	"go/ast"
	"unicode"
)

type Ioc struct {
	IsLazy     bool   `annotation:"name=lazy,default=true"`
	N          string `annotation:"name=name,default="`
	Alias      string `annotation:"name=alias,default="`
	Initialize string `annotation:"name=init,default="`
	Px         string `annotation:"name=proxy,default="`
	Qualifier  string `annotation:"name=qualifier,default="`
}

var _ M = (*Ioc)(nil)

func (Ioc) Name() string {
	return "ioc"
}

func (i Ioc) Match(node ast.Node) (err error) {
	if i.Initialize != "" && !unicode.IsUpper([]rune(i.Initialize)[0]) {
		err = fmt.Errorf("the `@Ioc(init)` value needs to start with a capital case")
		return
	}

	if _, ok := node.(*ast.FuncDecl); !ok {
		err = fmt.Errorf("the position of the `@Ioc` annotation is incorrect, needed is function (ast.FuncDecl)")
	}
	return
}

func (i Ioc) As() (_ M) {
	return
}
