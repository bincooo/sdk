package core

import (
	"encoding/json"
	"errors"
	"fmt"
	annotation "github.com/YReshetko/go-annotation/pkg"
	annotations "github.com/bincooo/sdk/gen/annotation"
	"os/exec"
	"reflect"
	"regexp"
	"strings"

	. "github.com/bincooo/sdk/stream"
)

type Builder func(proc *Processor) map[string][]byte

type Processor struct {
	wire     string
	builders map[string]Builder
	mapping  map[annotation.Node][]Convertor
}

var _ annotation.AnnotationProcessor = (*Processor)(nil)

var (
	proc *Processor

	importPathMap = make(map[string]Imported)
)

func init() {
	proc = &Processor{
		builders: make(map[string]Builder),
		mapping:  make(map[annotation.Node][]Convertor),
	}

	annotation.Register[annotations.Gen](proc)
	annotation.Register[annotations.Ioc](proc)
	annotation.Register[annotations.Proxy](proc)
	annotation.Register[annotations.Router](proc)
}

func Alias[T any]() {
	annotation.Register[T](proc)
}

func (proc *Processor) Version() string {
	return "v1.0.0"
}

func (proc *Processor) Name() string {
	return "IoC"
}

func (proc *Processor) Process(node annotation.Node) error {
	return errors.Join(
		scanAnnotated[annotations.Gen](proc, node, func(t annotations.Gen) Builder {
			proc.wire = t.Target
			return nil
		}),
		scanAnnotated[annotations.Ioc](proc, node, func(tag annotations.Ioc) Builder {
			return Ioc
		}),
		scanAnnotated[annotations.Router](proc, node, func(tag annotations.Router) Builder {
			return Router
		}),
		scanAnnotated[annotations.Proxy](proc, node, func(tag annotations.Proxy) Builder {
			return Px
		}),
	)
}

func (proc *Processor) Output() (ops map[string][]byte) {
	ops = make(map[string][]byte)
	for n, builder := range proc.builders {
		fmt.Println("build tag: ", n)
		for k, v := range builder(proc) {
			ops[k] = v
		}
	}
	return
}

func scanAnnotated[T annotations.M](proc *Processor, node annotation.Node, then func(t T) Builder) (err error) {
	var zero T
	slice := FindAnnotations[T](node.Annotations())
	if len(slice) == 0 {
		return
	}

	if len(slice) > 1 {
		to := reflect.TypeOf(zero)
		err = fmt.Errorf("expected 1 `%s` annotation, but got: %d", to.String(), len(slice))
		return
	}

	goAst := node.ASTNode()
	zero = slice[0]
	if err = zero.Match(goAst); err != nil {
		return
	}

	importPath, ok := importPathMap[node.Meta().Dir()]
	if !ok {
		importPath = Imported{}
		importPath.Alias, importPath.ImportPath, err = commandAsImportPath(node.Meta().Dir())
		if err != nil {
			return
		}
		importPathMap[node.Meta().Dir()] = importPath
	}

	convertor := newConvertor(zero, goAst, importPath.ImportPath)
	proc.mapping[node] = append(proc.mapping[node], convertor)
	if then != nil {
		if _, ok = proc.builders[zero.Name()]; !ok {
			if builder := then(zero); builder != nil {
				proc.builders[zero.Name()] = builder
			}
		}
	}
	return
}

func FindAnnotations[T any](a []annotation.Annotation) []T {
	return Map(OfSlice(a).Filter(ofType[T]), toType[T]).ToSlice()
}

func ofType[T any](a annotation.Annotation) bool {
	if m, ok := a.(annotations.M); ok {
		for {
			if n := m.As(); n != nil {
				m = n
			} else {
				_, ok = m.(T)
				return ok
			}
		}
	}

	_, ok := a.(T)
	return ok
}

func toType[T any](a annotation.Annotation) (t T) {
	if m, ok := a.(annotations.M); ok {
		for {
			if n := m.As(); n != nil {
				m = n
			} else {
				return m.(T)
			}
		}
	}

	return a.(T)
}

func commandAsImportPath(dir string) (alias, importPath string, err error) {
	command := exec.Command("go", "list", "-json", "-find", dir)
	output, err := command.Output()
	if err != nil {
		return
	}

	var obj map[string]interface{}
	if err = json.Unmarshal(output, &obj); err != nil {
		return
	}

	alias = obj["Name"].(string)
	importPath = obj["ImportPath"].(string)
	return
}

func ToSnakeCase(str string) (value string) {
	re := regexp.MustCompile("([A-Z])")
	snakeCase := re.ReplaceAllString(str, "_$1")
	snakeCase = strings.TrimPrefix(snakeCase, "_")
	value = strings.ToLower(snakeCase)
	return
}
