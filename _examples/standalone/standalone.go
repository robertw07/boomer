package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/myzhan/boomer"
)

func foo1() {
	start := time.Now()
	n := rand.Intn(50)
	time.Sleep(time.Millisecond * time.Duration(n))
	elapsed := time.Since(start)

	// Report your test result as a success, if you write it in python, it will looks like this
	// events.request_success.fire(request_type="http", name="foo", response_time=100, response_length=10)
	globalBoomer.RecordSuccess("http", "foo1", elapsed.Nanoseconds()/int64(time.Millisecond), int64(10))
}

func foo2() {
	start := time.Now()
	n := rand.Intn(50)
	time.Sleep(time.Millisecond * time.Duration(n))
	elapsed := time.Since(start)

	// Report your test result as a success, if you write it in python, it will looks like this
	// events.request_success.fire(request_type="http", name="foo", response_time=100, response_length=10)
	if n > 40 {
		globalBoomer.RecordFailure("http", "foo2", elapsed.Nanoseconds()/int64(time.Millisecond), "")
	} else {
		globalBoomer.RecordSuccess("http", "foo2", elapsed.Nanoseconds()/int64(time.Millisecond), int64(10))
	}
}

func foo3() {
	start := time.Now()
	n := rand.Intn(50)
	time.Sleep(time.Millisecond * time.Duration(n))
	elapsed := time.Since(start)

	// Report your test result as a success, if you write it in python, it will looks like this
	// events.request_success.fire(request_type="http", name="foo", response_time=100, response_length=10)
	globalBoomer.RecordSuccess("http", "foo3", elapsed.Nanoseconds()/int64(time.Millisecond), int64(10))
}

func foo4() {
	start := time.Now()
	n := rand.Intn(50)
	time.Sleep(time.Millisecond * time.Duration(n))
	elapsed := time.Since(start)

	// Report your test result as a success, if you write it in python, it will looks like this
	// events.request_success.fire(request_type="http", name="foo", response_time=100, response_length=10)
	globalBoomer.RecordSuccess("http", "foo4", elapsed.Nanoseconds()/int64(time.Millisecond), int64(10))
}

var globalBoomer *boomer.Boomer

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	task1 := &boomer.Task{
		Name:   "foo1",
		Weight: 10,
		Fn:     foo1,
	}

	task2 := &boomer.Task{
		Name:   "foo2",
		Weight: 10,
		Fn:     foo2,
	}

	task3 := &boomer.Task{
		Name:   "foo3",
		Weight: 20,
		Fn:     foo3,
	}

	task4 := &boomer.Task{
		Name:   "foo3",
		Weight: 10,
		Fn:     foo4,
	}

	numClients := 30
	spawnRate := float64(1)
	globalBoomer = boomer.NewStandaloneBoomer(numClients, spawnRate)
	globalBoomer.AddOutput(boomer.NewConsoleOutputWithOptions(&boomer.OutputOptions{
		PercentTime: 95,
	}))
	globalBoomer.Run(task1, task2, task3, task4)
}
