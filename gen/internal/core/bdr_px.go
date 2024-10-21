package core

import (
	"bytes"
	"fmt"
	"github.com/bincooo/sdk/gen/annotation"
	"go/ast"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	. "github.com/bincooo/sdk/stream"
)

var (
	pxTemplate0 = `package {{ .package }}

import (
	"github.com/bincooo/sdk"
)

{{ .code }}
`
	pxTemplate1 = `
type _{{ .name }} struct {
	proto {{ .name }}
}

func {{ .name }}Px(proto {{ .name }}) {{ .name }} {
	return &_{{ .name }}{proto}
}

`
)

func Px(proc *Processor) (ops map[string][]byte) {
	ops = make(map[string][]byte)
	for node, converters := range proc.mapping {
		for _, convert := range converters {
			if !convert.As("proxy") {
				continue
			}

			n := convert.GetAstName()
			instance, err := template.New(n).Parse(pxTemplate1)
			if err != nil {
				panic(err)
			}

			var buf bytes.Buffer
			if err = instance.Execute(&buf, map[string]string{
				"name": n,
			}); err != nil {
				panic(err)
			}

			spec := convert.node.(*ast.TypeSpec)
			methods := spec.Type.(*ast.InterfaceType).Methods
			for _, method := range methods.List {

				var (
					pos = 1
				)

				argNames := make([]string, 0)
				extractArguments := convert.ExtractArguments(node.Lookup(), method)
				args := strings.Join(FlatMap(OfSlice(extractArguments), func(t Argv) []string {
					return Map(OfSlice(t.Names), func(n string) string {
						if n == "" || n == "_" {
							n = "var" + strconv.Itoa(pos)
							pos++
						}
						argNames = append(argNames, n)
						return n + " " + t.String()
					}).ToSlice()
				}).ToSlice(), ", ")

				returnNames := make([]string, 0)
				extractReturns := convert.ExtractReturns(node.Lookup(), method)
				returns := strings.Join(FlatMap(OfSlice(extractReturns), func(t Argv) []string {
					return Map(OfSlice(t.Names), func(n string) string {
						if n == "" || n == "_" {
							n = "var" + strconv.Itoa(pos)
							pos++
						}
						returnNames = append(returnNames, n)
						return n + " " + t.String()
					}).ToSlice()
				}).ToSlice(), ", ")
				if returns != "" {
					returns = "(" + returns + ")"
				}

				buf.WriteString(fmt.Sprintf(`func (obj *_%s) %s(%s) %s {`, n, method.Names[0].String(), args, returns))
				buf.WriteString(fmt.Sprintf(`
				var ctx = &sdk.Context{
					Name:     "%s",
					Receiver: obj.proto,
					In:       []any{%s},
					Out:      []any{%s},
				}`, n, strings.Join(argNames, ", "), strings.Join(returnNames, ", ")))

				pos = 0
				args = strings.Join(FlatMap(OfSlice(extractArguments), func(t Argv) []string {
					return Map(OfSlice(t.Names), func(_ string) (str string) {
						str = fmt.Sprintf("ctx.In[%d].(%s)", pos, t.Interface.String())
						pos++
						return
					}).ToSlice()
				}).ToSlice(), ", ")

				pos = 0
				returns = strings.Join(Map(OfSlice(returnNames), func(n string) (str string) {
					str = fmt.Sprintf("ctx.Out[%d] = %s", pos, n)
					return
				}).ToSlice(), "\n")

				vars := strings.Join(returnNames, ", ")
				if vars != "" {
					vars = vars + " = "
				}

				buf.WriteString(fmt.Sprintf(`
				ctx.Do = func() {
					%sobj.proto.%s(%s)
					%s
				}`, vars, method.Names[0].String(), args, returns))

				// TODO -
				buf.WriteString(fmt.Sprintf("\n\t%s(ctx)\n", convert.tag.(annotation.Proxy).Target))

				pos = 0
				returns = strings.Join(FlatMap(OfSlice(extractReturns), func(t Argv) []string {
					return Map(OfSlice(t.Names), func(_ string) (str string) {
						str = fmt.Sprintf("ctx.Out[%d].(%s)", pos, t.Interface.String())
						pos++
						return
					}).ToSlice()
				}).ToSlice(), ", ")
				if returns != "" {
					returns = "return " + returns
				}

				buf.WriteString(fmt.Sprintf("\n\t %s }\n\n", returns))
			}

			instance, err = template.New(n).Parse(pxTemplate0)
			if err != nil {
				panic(err)
			}

			var buf1 bytes.Buffer
			if err = instance.Execute(&buf1, map[string]string{
				"package": node.Meta().PackageName(),
				"code":    buf.String(),
			}); err != nil {
				panic(err)
			}

			ops[filepath.Join(node.Meta().Dir(), ToSnakeCase(n)+"_px.gen.go")] = buf1.Bytes()
		}
	}
	// TODO
	return
}
