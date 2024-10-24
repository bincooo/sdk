package core

import (
	"bytes"
	"fmt"
	annotation "github.com/YReshetko/go-annotation/pkg"
	annotations "github.com/bincooo/sdk/gen/annotation"
	"go/ast"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/template"

	. "github.com/bincooo/sdk/stream"
)

type Imported struct {
	Alias      string
	ImportPath string
}

type pxInfo struct {
	ip   Imported
	name string
}

var (
	iocTemplate = `package {{ .package }}

import (
	"github.com/bincooo/sdk"
{{- range $import := .imports}}
	{{$import.String}}
{{- end}}
)

func Injects(container *sdk.Container) error {
	// Registered container
	//
{{- range $code := .codes}}
	{{$code}}
{{ end }}

	return nil
}
`
)

func (px pxInfo) String() string {
	if len(px.ip.Alias) == 0 {
		return fmt.Sprintf(`"%s"`, px.name)
	}
	return fmt.Sprintf(`%s.%s`, px.ip.Alias, px.name)
}

func (ip Imported) String() string {
	if len(ip.Alias) == 0 {
		return fmt.Sprintf(`"%s"`, ip.ImportPath)
	}
	return fmt.Sprintf(`%s "%s"`, ip.Alias, ip.ImportPath)
}

func Ioc(proc *Processor) (ops map[string][]byte) {
	var (
		pkg     = genPackageName(proc.wire)
		imports []Imported
		codes,
		activated []string
	)

	pxList, err := findPxList(proc.mapping)
	if err != nil {
		panic(err)
	}

	for node, convertors := range proc.mapping {
		for _, convert := range convertors {
			if !convert.As("ioc") {
				continue
			}

			// validate
			returns := convert.ExtractReturns(node.Lookup(), convert.node)
			{
				types := FlatMap(OfSlice(returns),
					func(re Argv) []string {
						return Map(OfSlice(re.Names), func(string) string {
							return re.Interface.String()
						}).ToSlice()
					}).ToSlice()
				size := len(types)
				if size == 0 || size > 2 {
					panic("the return value must provide >= 1 & <= 2")
				}
				if types[0] == "error" {
					panic("the return [1] value must provide object")
				}
				if size == 2 && types[1] != "error" {
					panic("the return [1] value must provide error")
				}
			}

			var buf strings.Builder
			meta := node.Meta()
			alias := meta.PackageName()
			imports, alias = Import(imports, alias, convert.ImportPath())

			importPath := returns[0].ImportPath
			if importPath == "" {
				importPath = convert.ImportPath()
				returns[0].Alias(alias)
				convert.Alias(alias)
			} else {
				if imported, ok := importPathMap[meta.Dir()]; ok {
					convert.Alias(imported.Alias)
				}
			}

			iocClass := Or(convert.tag.(annotations.Ioc).N == "", importPath+"."+returns[0].Interface.Ext(), convert.tag.(annotations.Ioc).N)
			results, padding := joinReturn(returns)
			if !convert.tag.(annotations.Ioc).IsLazy {
				activated = append(activated, fmt.Sprintf("_, err = sdk.InvokeBean[%s](container, \"%s\")\n\tif err != nil {\n\t\treturn err\n\t}", returns[0].String(), iocClass))
			}

			// 组件开启了代理
			if proxy := convert.tag.(annotations.Ioc).Px; proxy != "" {
				ok := false
				for _, px := range pxList {
					if pxClass := px.ip.ImportPath + "." + px.name; slices.Contains([]string{
						proxy,
						filepath.Join(filepath.Dir(convert.ImportPath()), proxy),
					}, pxClass) {
						imports, _ = Import(imports, px.ip.Alias, px.ip.ImportPath)
						buf.WriteString(fmt.Sprintf("sdk.Proxy[%s](container, \"%s\", %sPx)\n", px.String(), iocClass, px.String()))
						ok = true
						break
					}
				}

				if !ok {
					panic("not found proxy target: " + proxy)
				}
			}

			pos := 1
			// 组件分配别名
			if n := convert.tag.(annotations.Ioc).Alias; n != "" {
				buf.WriteString(fmt.Sprintf("container.Alias(\"%s\", \"%s\")\n", n, iocClass))
			}
			buf.WriteString(fmt.Sprintf("sdk.ProvideBean(container, \"%s\", func() (%s) {\n", iocClass, results))
			{
				// 参数生成
				var vars []string
				i := -1
				args := convert.ExtractArguments(node.Lookup(), convert.node)
				for _, argv := range args {
					for _, n := range argv.Names {
						i++
						if n == "" || n == "_" {
							n = "var" + strconv.Itoa(pos)
							pos++
						}

						vars = append(vars, n)
						importPath = argv.ImportPath
						if importPath == "" {
							importPath = convert.ImportPath()
							argv.Alias(alias)
						}

						iocClass = importPath + "." + argv.Interface.Ext()
						// 别名匹配
						if qualifier := convert.tag.(annotations.Ioc).Qualifier; qualifier != "" {
							values := strings.Split(qualifier, ",")
							for _, value := range values {
								value = strings.TrimSpace(value)
								idx := strings.Index(value, "]:")
								if value[0] != '[' || idx == -1 {
									break
								}

								nums := strings.Split(value[1:idx], "~")
								n1, n2 := -1, -1
								n1, err = strconv.Atoi(nums[0])
								if err != nil {
									panic("qualifier value parse to int err: " + err.Error())
								}
								if len(nums) > 1 {
									n2, err = strconv.Atoi(nums[1])
									if err != nil {
										panic("qualifier value parse to int err: " + err.Error())
									}
								} else {
									n2 = n1
								}

								ok := false
								for num := n1; num <= n2; num++ {
									if i == num {
										iocClass = value[idx+2:]
										ok = true
										break
									}
								}
								if !ok {
									break
								}
							}
						}

						buf.WriteString(fmt.Sprintf(`	%s, err := sdk.InvokeBean[%s](container, "%s")`, n, argv.String(), iocClass))
						buf.WriteString("\n")
						buf.WriteString(fmt.Sprintf("	if err != nil {\n		return %s, err\n	}", Or(returns[0].IsPointer, "nil", returns[0].String()+"{}")))
						buf.WriteString("\n")
					}
				}
				results = strings.Join(vars, ", ")
			}

			var1 := ""
			str := strings.Join(FlatMap(OfSlice(returns), func(t Argv) []string {
				return Map(OfSlice(t.Names), func(n string) string {
					if n == "" || n == "_" {
						n = "var" + strconv.Itoa(pos)
						pos++
					}
					if var1 == "" {
						var1 = n
					}
					return n
				}).ToSlice()
			}).ToSlice(), ", ")

			buf.WriteString(fmt.Sprintf("	%s := %s(%s)\n", str, convert.GetAstName(), results))
			// 执行初始化方法
			if init := convert.tag.(annotations.Ioc).Initialize; init != "" {
				buf.WriteString("	// Invoke initialize method\n")
				buf.WriteString(fmt.Sprintf("	%s.%s()\n", var1, init))
			}
			buf.WriteString(fmt.Sprintf("	return %s%s })", str, Or(padding, ", nil", "")))
			codes = append(codes, buf.String())
		}
	}

	if len(activated) > 0 {
		codes = append(codes, "\t// Initialized instance\n\t//\n\tvar err error")
		codes = append(codes, activated...)
	}

	instance, err := template.New("ioc").Parse(iocTemplate)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	data := map[string]interface{}{
		"package": pkg,
		"imports": imports,
		"codes":   codes,
	}

	if err = instance.Execute(&buf, data); err != nil {
		panic(err)
	}

	ops = make(map[string][]byte)
	ops[filepath.Join(proc.wire, "container.gen.go")] = buf.Bytes()
	return
}

func findPxList(mapping map[annotation.Node][]Convertor) (px []pxInfo, err error) {
	for node, converters := range mapping {
		for _, convert := range converters {
			if !convert.As("proxy") {
				continue
			}

			spec := convert.node.(*ast.TypeSpec)
			importPath, ok := importPathMap[node.Meta().Dir()]
			if !ok {
				importPath = Imported{}
				importPath.Alias, importPath.ImportPath, err = commandAsImportPath(node.Meta().Dir())
				if err != nil {
					return
				}
				importPathMap[node.Meta().Dir()] = importPath
			}

			px = append(px, pxInfo{
				name: spec.Name.Name,
				ip:   importPath,
			})
		}
	}
	return
}

func genPackageName(path string) string {
	_, importPath, err := commandAsImportPath(path)
	if err != nil {
		importPath = strings.ReplaceAll(filepath.Base(path), "-", "_")
	}
	return importPath
}

func joinReturn(returns []Argv) (str string, padding bool) {
	results := FlatMap(OfSlice(returns), func(re Argv) []string {
		return Map(OfSlice(re.Names), func(string) string {
			return re.String()
		}).ToSlice()
	}).ToSlice()

	if len(results) == 1 {
		results = append(results, "error")
		padding = true
	}

	str = strings.Join(results, ", ")
	return
}

func Import(imports []Imported, alias, importPath string) ([]Imported, string) {
	pos := 1
	has := false
	change := false
	for _, ip := range imports {
		if ip.ImportPath == importPath {
			has = true
			break
		}

		if alias == "_" {
			continue
		}

		if ip.Alias == alias {
			change = true
			alias += strconv.Itoa(pos)
			pos++
		}
	}

	if has {
		return imports, alias
	}

	return append(imports, Imported{Or(change, alias, ""), importPath}), alias
}

func Or[T any](expr bool, a1 T, a2 T) T {
	if expr {
		return a1
	} else {
		return a2
	}
}
