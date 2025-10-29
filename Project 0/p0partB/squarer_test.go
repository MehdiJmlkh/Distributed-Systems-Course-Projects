// Tests for SquarerImpl. Students should write their code in this file.

package p0partB

import (
	"fmt"
	"testing"
	"time"
)

const (
	timeoutMillis = 5000
)

func testBasic(t *testing.T, name string, inputNum int, expected int) {
	fmt.Println("Test", name)
	input := make(chan int)
	sq := SquarerImpl{}
	squares := sq.Initialize(input)
	go func() {
		input <- inputNum
	}()
	timeoutChan := time.After(time.Duration(timeoutMillis) * time.Millisecond)
	select {
	case <-timeoutChan:
		t.Error("Test timed out.")
	case result := <-squares:
		if result != expected {
			t.Errorf("Error, got result %d, expected %d (=%d^2).", result, expected, inputNum)
		}
	}
}

func TestSquarePositiveNumber(t *testing.T) {
	testBasic(t, "square positive number", 4, 16)
}

func TestSquareNegativeNumber(t *testing.T) {
	testBasic(t, "square negative number", -5, 25)
}

func TestSquareZero(t *testing.T) {
	testBasic(t, "square zero", 0, 0)
}

func TestSquareOne(t *testing.T) { // identity case
	testBasic(t, "square one", 1, 1)
}

func TestSquareNegativeOne(t *testing.T) {
	testBasic(t, "square negative one", -1, 1)
}

func TestMultipleInputs(t *testing.T) {
	input := make(chan int)
	sq := SquarerImpl{}
	squares := sq.Initialize(input)
	defer sq.Close()

	go func() {
		for i := -50; i < 50; i++ {
			input <- i
		}
	}()

	expecteds := make(chan int, 100)
	for i := -50; i < 50; i++ {
		expecteds <- i * i
	}

	timeoutChan := time.After(time.Duration(timeoutMillis) * time.Millisecond)
	for {
		select {
		case <-timeoutChan:
			t.Error("Test timed out.")
		case result := <-squares:
			fmt.Println("Result:", result)
			popped := <-expecteds
			if result != popped {
				t.Errorf("Error, got result %d, expected %d.", result, popped)
			}
		}
		if len(expecteds) == 0 {
			break
		}
	}
}
