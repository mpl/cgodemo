package main

// #include <stdlib.h>
import "C"
import "fmt"

func main() {
	cs := C.CString("Hello world")
	s := C.GoString(cs)
	fmt.Println(s)
}
