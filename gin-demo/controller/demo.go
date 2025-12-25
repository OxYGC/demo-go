package main

import (
	"fmt"
	"sync"
	"time"
)

/*
*
WaitGroup.Add(n)
wg.Done()（通常用 defer）
wg.Wait() 等所有协程结束
*/
func main() {

}

func producer(ch chan<- int) {
	for i := 0; i < 10; i++ {
		ch <- i
	}
	close(ch)
}

func consumer(id int, ch <-chan int, wg *sync.WaitGroup) {
	defer wg.Done()
	for n := range ch {
		fmt.Println(id, n)
		// k slow process
		time.Sleep(2 * time.Millisecond)
	}
	fmt.Println("work done", id)
}

type Task struct {
	Id        int
	Timestamp time.Time
	Data      string
}

type User struct {
	Id   int
	Name string
	Age  int
}

type Deposit struct {
	Hash      string
	From      string
	To        string
	Amount    string
	Timestamp int64
}
