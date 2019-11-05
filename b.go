package b

import "os"

//go:noinline
func a(ch chan int) int {
	sl := make([]int, 4096)
	sl[4095] = 127

	ch <- sl[1] + sl[4095]

	go func(ch chan<- int) {
		ch <- 1
	}(ch)

	return <-ch
}

type A struct {
	a, b, c, d, e int32
}

//go:noinline
func Foo(a A, x, y int32) int32 {
	return a.a + a.b + a.c + a.d + a.e + x + y
}

func Caller() {
	ch := make(chan int, 1)

	os.Exit(a(ch))
}
