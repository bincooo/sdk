package mod

import (
	"fmt"
	"github.com/bincooo/sdk"
	"github.com/gin-gonic/gin"
)

type A struct{}

// @Router(path="/B")
type B struct {
	*A
}

// @Proxy(target="Test")
type Hi interface {
	Echo(i int) error
	Hi()
}

func Test(ctx *sdk.Context) {
	fmt.Println("Test: ", ctx.Name)
}

// @Ioc(name="tests.A", proxy="github.com/bincooo/sdk/tests.Hi")
func NewA() *A {
	return &A{}
}

// @Ioc(lazy="false", alias="tests.B", qualifier="[0~1]:tests.A")
func NewB(_, a1 *A) (B, error) {
	return B{a1}, nil
}

func (A) Echo() {
	fmt.Println("A.Echo()")
}

// @Router(method="POST", path="echo")
func (b *B) Echo(gtx *gin.Context) {
	b.A.Echo()
	fmt.Println("B.Echo()")
}

// @DELETE(path="/find")
func (b *B) Find(gtx *gin.Context) {
	fmt.Println("B.Find()")
}
