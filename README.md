## 注解生成 & ioc 容器库

### 使用Ioc

```golang
// model.go
type A struct {
	
}

type B struct {
	*A
}

// @Ioc(lazy="false")
func NewA() *A {
	return &A{}
}

// @Ioc()
func NewB(a *A) *B, error {
	return &B{a}, nil
}

// main.go
import (
	"github.com/bincooo/sdk"
)
// @Gen(target="wire")
func main() {
    container := sdk.NewContainer()
	err := wire.Injects(container)
    if err != nil {
        panic(err)
    }

    err = container.Run()
	if err != nil {
        panic(err)
    }
}

// cmd/main.go
import (
	"github.com/bincooo/sdk/gen"
)
func main() {
	gen.Process()
}
```

### 使用代理
```go
// model.go
type A struct {
	
}

// @Proxy(target="invoke")
type EchoI interface {
	Echo()
}

func invoke(ctx sdk.Context) {
	// before
	fmt.Println(ctx.In...)
	
	// instance & method name
	fmt.Println(ctx.Receiver, ctx.Name)
	
	// do method
	ctx.Do()
	
	// after
	fmt.Println(ctx.Out...)
}

func (A) Echo() {
	fmt.Println("A.Echo()")
}

// @Ioc(name="model.A", lazy="false", proxy="model.Echo")
func NewA() *A {
    return &A{}
}

// main.go
import (
"github.com/bincooo/sdk"
)
// @Gen(target="wire")
func main() {
    container := sdk.NewContainer()
    err := wire.Injects(container)
    if err != nil {
        panic(err)
    }

    err = container.Run()
    if err != nil {
        panic(err)
    }

    bean, err := sdk.InvokeAs[model.Echo](container, "model.A")
    if err != nil {
        panic(err)
    }
    bean.Echo()
}

```

执行命令生成代码：

```shell
go run cmd/main.go
```


### 参考示例

1. [tests](tests/model.go)
2. [sdk-examples](https://github.com/bincooo/sdk-examples)