package core

import (
	"bytes"
	"fmt"
	annotation "github.com/YReshetko/go-annotation/pkg"
	annotations "github.com/bincooo/sdk/gen/annotation"
	"go/ast"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	routeTemplate = `package {{ .package }}

import (
	"github.com/gin-gonic/gin"
)

{{ range $key, $code := .codes}}
func ({{ $key }}) Routers(route gin.IRouter) {
{{- range $line := $code }}
	{{ $line }}{{ end }}
}
{{ end }}
`
)

type info struct {
	dir string
	pkg string
}

func Router(proc *Processor) (ops map[string][]byte) {
	structures := ExtractStructure(proc.mapping)
	cached := make(map[info]map[string][]string)
	for node, converters := range proc.mapping {
		var codes map[string][]string
		for _, convert := range converters {
			if !convert.As("router") {
				continue
			}

			var isPointer bool
			var receiver, routeBaseDir, method string
			{
				fd, ok := convert.node.(*ast.FuncDecl)
				if !ok {
					continue
				}
				receiver = annotations.MethodReceiver(fd)
				structure, ok := structures[receiver]
				if ok {
					routeBaseDir = structure.tag.(annotations.Router).Path
				}
				method = fd.Name.Name
				_, ok = fd.Recv.List[0].Type.(*ast.StarExpr)
				if ok {
					isPointer = true
				}
			}

			var1 := Or(receiver == "obj", "obj1", "obj")
			var buf bytes.Buffer
			buf.WriteString(fmt.Sprintf("route.%s(\"%s\", %s.%s)",
				strings.ToUpper(convert.tag.(annotations.Router).Method),
				filepath.Join(routeBaseDir, convert.tag.(annotations.Router).Path),
				var1,
				method),
			)

			if codes == nil {
				codes = make(map[string][]string)
			}

			var1 += " " + Or(isPointer, "*", "") + receiver
			codes[var1] = append(codes[var1], buf.String())
		}

		if codes == nil {
			continue
		}

		// 合并归类代码
		i := info{node.Meta().Dir(), node.Meta().PackageName()}
		if cache, ok := cached[i]; ok {
			for key, value := range codes {
				if _, ok = cache[key]; ok {
					cache[key] = append(cache[key], value...)
					continue
				}
				cache[key] = value
			}
			cached[i] = cache
			continue
		}
		cached[i] = codes
	}

	for i, codes := range cached {
		instance, err := template.New("router").Parse(routeTemplate)
		if err != nil {
			panic(err)
		}

		var buf bytes.Buffer
		data := map[string]interface{}{
			"package": i.pkg,
			"codes":   codes,
		}

		if err = instance.Execute(&buf, data); err != nil {
			panic(err)
		}

		ops = make(map[string][]byte)
		ops[filepath.Join(i.dir, "router.gen.go")] = buf.Bytes()
	}
	return
}

func ExtractStructure(mapping map[annotation.Node][]Convertor) (re map[string]Convertor) {
	re = make(map[string]Convertor)
	for _, converters := range mapping {
		for _, convert := range converters {
			if !convert.As("router") {
				continue
			}

			var receiver string
			if node, ok := convert.node.(*ast.TypeSpec); !ok {
				continue
			} else {
				receiver = node.Name.Name
				re[receiver] = convert
			}
		}
	}
	return
}
