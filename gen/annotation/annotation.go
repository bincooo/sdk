package annotation

import "go/ast"

type M interface {
	Name() string
	Match(node ast.Node) error
	As() M
}
