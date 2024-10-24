package sdk

import (
	"errors"
	"fmt"
	"github.com/bincooo/sdk/runtime"
	"github.com/samber/do/v2"
	"os"
	"os/signal"
	"reflect"
	run "runtime"
	"slices"
	"strings"
	"sync"
)

type keys struct {
	sync.Mutex
	g []string
}

type typeWarp struct {
	t      reflect.Type
	invoke func(any) (any, error)
}

type Initializer interface {
	Init(*Container) error
	Order() int
}

type singleInitializer struct {
	order int
	init  func(*Container) error
}

type Container struct {
	inject *do.RootScope
	alias  map[string]string
	types  map[string]typeWarp

	registers []string

	px map[string][]typeWarp
}

var (
	threadLocal = runtime.NewThreadLocal[*keys](func() *keys {
		return &keys{}
	})
)

func (k *keys) push(key string) (re bool) {
	k.Lock()
	defer k.Unlock()

	if !slices.Contains(k.g, key) {
		re = true
	}

	k.g = append(k.g, key)
	return
}

func (i singleInitializer) Init(container *Container) (err error) {
	if i.init == nil {
		return
	}
	return i.init(container)
}

func (i singleInitializer) Order() int {
	return i.order
}

func NewContainer() *Container {
	return &Container{
		inject: do.New(),
		alias:  make(map[string]string),
		types:  make(map[string]typeWarp),

		px: make(map[string][]typeWarp),
	}
}

func InitializedWrapper(order int, init func(*Container) error) Initializer {
	return &singleInitializer{order, init}
}

func (c *Container) Run(signals ...os.Signal) (err error) {
	beans, err := ListInvokeAs[Initializer](c)
	if err != nil {
		return
	}

	slices.SortFunc[[]Initializer](beans, func(a, b Initializer) int {
		return or(a.Order() == b.Order(), 0, or(a.Order() > b.Order(), 1, -1))
	})

	for _, bean := range beans {
		if err = bean.Init(c); err != nil {
			return
		}
	}

	// TODO -
	if len(signals) > 0 {
		w := make(chan os.Signal, 1)
		signal.Notify(w, signals...)
		<-w
	}
	return
}

func (c *Container) Inject() *do.RootScope {
	return c.inject
}

func (c *Container) Alias(name, fullName string) {
	if n, ok := c.alias[name]; ok {
		panic("alias '" + n + "' already exists")
	}
	c.alias[name] = fullName
}

func (c *Container) HealthLogger() string {
	injector := do.ExplainInjector(c.inject)
	return injector.String()
}

func (c *Container) Stop() (err error) {
	err = c.inject.Shutdown()
	if err != nil {
		return
	}

	// TODO
	return
}

func ProvideBean[T any](container *Container, name string, provider func() (T, error)) {
	container.registers = append(container.registers, name)
	var zero T
	ox := reflect.TypeOf(zero)
	if ox == nil {
		ox = reflect.TypeOf((*T)(nil))
		ox = ox.Elem()
	}
	// if ox.Kind() == reflect.Ptr {
	// 	ox = ox.Elem()
	// }

	container.types[name] = typeWarp{
		t: ox,
		invoke: func(any) (any, error) {
			return InvokeBean[T](container, name)
		},
	}
	do.ProvideNamed[T](container.inject, name, func(i do.Injector) (T, error) {
		return provider()
	})
}

func OverrideBean[T any](container *Container, name string, provider func() (T, error)) {
	container.registers = append(container.registers, name)
	var zero T
	ox := reflect.TypeOf(zero)
	if ox == nil {
		ox = reflect.TypeOf((*T)(nil))
		ox = ox.Elem()
	}
	//if ox.Kind() == reflect.Ptr {
	//	ox = ox.Elem()
	//}

	container.types[name] = typeWarp{
		t: ox,
		invoke: func(any) (any, error) {
			return InvokeBean[T](container, name)
		},
	}
	do.OverrideNamed[T](container.inject, name, func(i do.Injector) (T, error) {
		return provider()
	})
}

func InvokeBean[T any](container *Container, name string) (t T, err error) {
	for {
		if n, ok := container.alias[name]; ok {
			name = n
		} else {
			break
		}
	}

	var zero T
	if !threadLocal.Ex() {
		defer threadLocal.Remove()
	}

	// checked
	value := threadLocal.Load()
	if !value.push(name) {
		return zero, warpError(fmt.Errorf("acircular dependency occurs:\n%s", join(value.g, name)))
	}

	t, err = do.InvokeNamed[T](container.inject, name)
	return
}

func InvokeAlias[T any](container *Container, name string) (T, error) {
	for {
		if n, ok := container.alias[name]; ok {
			name = n
		} else {
			break
		}
	}

	var zero T
	warp, ok := container.types[name]
	if !ok {
		return zero, fmt.Errorf("the alias '%s' does not exist", name)
	}

	ox := reflect.TypeOf(zero)
	if ox == nil {
		ox = reflect.TypeOf((*T)(nil))
	}

	if ox.Kind() == reflect.Ptr {
		ox = ox.Elem()
	}
	if warp.t.Implements(ox) {
		bean, err := warp.invoke(nil)
		if err != nil {
			return zero, err
		}

		if zero, ok = bean.(T); ok {
			return zero, nil
		}
	}

	return zero, fmt.Errorf("type (T) does not conform to the implemented")
}

func ListInvokeAs[T any](container *Container) (re []T, err error) {
	var zero T
	ox := reflect.TypeOf(zero)
	if ox == nil {
		ox = reflect.TypeOf((*T)(nil))
	}

	if ox.Kind() == reflect.Ptr {
		ox = ox.Elem()
	}

	for _, name := range container.registers {
		if tw, ok := container.types[name]; ok {
			if !tw.t.Implements(ox) && tw.t != ox {
				continue
			}

			bean, ie := tw.invoke(nil)
			if ie != nil {
				return nil, ie
			}

			if zero, ok = bean.(T); ok {
				re = append(re, zero)
			}
		}
	}

	return
}

func InvokeAs[T any](container *Container, name string) (t T, err error) {
	var zero T
	if !threadLocal.Ex() {
		defer threadLocal.Remove()
	}

	// checked
	en := false
	value := threadLocal.Load()
	if name == "" {
		name = do.NameOf[T]()
		en = true
	}

	if en { // 空名称使用默认匹配
		if !value.push(name) {
			return zero, warpError(fmt.Errorf("acircular dependency occurs:\n%s", join(value.g, name)))
		}
		return do.InvokeAs[T](container.inject)
	}

	for {
		if n, ok := container.alias[name]; ok {
			name = n
		} else {
			break
		}
	}

	beans, err := ListInvokeAs[T](container)
	if err != nil {
		return zero, err
	}

	tw := container.types[name]
	for _, bean := range beans {
		if tw.t == reflect.TypeOf(bean) {
			if f, ok := isPx(container, name, reflect.TypeOf(bean)); ok {
				return f(bean).(T), nil
			}
			return bean, nil
		}
	}

	err = fmt.Errorf("no instance matching '%s' was found", name)
	return
}

func Proxy[T any](container *Container, name string, f func(T) T) {
	var zero T
	ox := reflect.TypeOf(zero)
	if ox == nil {
		ox = reflect.TypeOf((*T)(nil))
	}

	if ox.Kind() == reflect.Ptr {
		ox = ox.Elem()
	}

	container.px[name] = append(container.px[name], typeWarp{ox, func(a any) (any, error) {
		return f(a.(T)), nil
	}})
}

func isPx(container *Container, name string, ex reflect.Type) (f func(any) any, px bool) {
	// 是否代理
	if types, ok := container.px[name]; ok {
		for _, tx := range types {
			if ex.Implements(tx.t) || tx.t == ex {
				f = func(a any) (i any) {
					i, _ = tx.invoke(a)
					return
				}

				px = true
				return
			}
		}
	}
	return
}

func warpError(err error) error {
	if err != nil {
		count := -1
		frame := runtime.CallerFrame(func(fe run.Frame) (ok bool) {
			if count > -1 {
				count++
				return count >= 2 // 第x个栈
			}

			if i := len(fe.File); fe.File[i-12:] == "container.go" &&
				fe.Function == "github.com/bincooo/sdk.warpError" {
				count = 0 // 开始计数
			}
			return
		})

		if frame != nil {
			tr := strings.Split(frame.Function, ".")
			err = errors.Join(err, fmt.Errorf(`in %s # %s:%d`, frame.File, tr[:len(tr)-1], frame.Line))
		}
	}

	return err
}

func join(slice []string, n string) (str string) {
	idx := -1
	sliceL := len(slice)
	for i, it := range slice {
		if idx == -1 && it == n {
			idx = i
		}

		switch i {
		case idx:
			str += "╭- " + it + "\n"
		case sliceL - 1:
			str += "╰> " + it + "\n"
		default:
			if idx == -1 {
				str += "   " + it + "\n"
			} else {
				str += "|  " + it + "\n"
			}
		}
	}
	return
}

func or[T any](condition bool, a1, a2 T) T {
	if condition {
		return a1
	} else {
		return a2
	}
}
