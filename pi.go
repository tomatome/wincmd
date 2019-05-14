package main

import (
	"fmt"
	"time"
)

const cNumMax = 999999999

func main() {
	t1 := time.Now()

	pi := 1.0
	a := 1
	b := 1 / float64(a)
	for b > 1e-30 {
		a += 2
		b = 1 / float64(a)
		if a/2%2 == 0 {
			pi += b
		} else {
			pi -= b
		}
	}

	pi *= 4
	t2 := time.Now()
	fmt.Printf("PI = %v; Time = %.2f s\n", pi, t2.Sub(t1).Seconds())
}
