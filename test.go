package main

import (
	"fmt"
)

import "sync"
import "time"

var mu sync.Mutex

func f() {
	mu.Lock()
	defer mu.Unlock()
	// critical section
	fmt.Println("Function f is running")
}

func main() {
	// var numbers = [5]bool{true, true, false, false, true}
	// for i := range numbers {

	// ch := make(chan int, 10)
	// ch <- 10
	// for i := range ch {
	// 	fmt.Println(i)
	// }
	// k := 10
	// for k < 20 {
	// 	fmt.Println(k + 1)
	// 	k++
	// }
	// fmt.Println("End of first loop")
	// k = 10
	// for i := k + 1; i <= 20; i++ {
	// 	fmt.Println(i)
	// 	k++
	// }
	mu.Lock()
	f()
	fmt.Println("Main function is running")
	mu.Unlock()
	// Sleep for a while to allow goroutine to finish
	time.Sleep(1 * time.Second)
	// matchIndex := make([]int, 10)
	// fmt.Println("Match Index:", matchIndex)

	// }
}
