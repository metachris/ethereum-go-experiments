package main

import "fmt"

func test1() {
	fmt.Println(("test"))

	arr := [3]string{"a", "b", "c"}
	// arr := []bool{true, true, false}
	fmt.Println("arr", arr)

	b := arr[1:3]
	fmt.Println("b", b)

	arr[1] = "x"
	fmt.Println("arr", arr)
}

func test2() {
	for i := 1; i < 5; i++ {
		fmt.Println(i)
	}

	// fmt.Println("test2")
	// a := make([]int, 5)

	// a[1] = 7
	// printSlice("a", a)

	// for i, v := range a {
	// 	fmt.Println(i, v)
	// }
}

func main() {
	test2()
}

func printSlice(s string, x []int) {
	fmt.Printf("%s len=%d cap=%d %v\n", s, len(x), cap(x), x)
}
