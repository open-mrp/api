package main

type A struct{ X int }
type B struct{ X int }

func main() {
	var a A
	_ = B(a)
}
