package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/myzhan/boomer"
)

func foo1() {
	start := time.Now()
	//n := rand.Intn(50)
	time.Sleep(time.Millisecond * time.Duration(20))
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

func foo5() {
	//pool, _ := ants.NewPool(5000)
	//atomic.AddInt32(&r.numClients, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	for i := 1; i <= 5000; i++ {
		go func() {
			for {
				nr, _ := http.NewRequest("POST", "https://bsc-mainnet.bk.nodereal.cc/v1/f34f62e7c0b343ef9f5bf80031a49cc2",
					strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_getStorageAt\",\"params\":[\"0xbA2aE424d960c26247Dd6c32edC70B295c744C43\",\"0x0\",\"0x14da8d9\"]}"))
				nr.Header.Add("Content-Type", "application/json")

				client := &http.Client{Timeout: 60 * time.Second}
				sTime := time.Now()
				_, err := client.Do(nr)
				eTime := time.Now()
				duration := eTime.Sub(sTime).Milliseconds()
				a := rand.Intn(100)
				if a == 50 {
					fmt.Println("******", duration, "******", err)
				}
			}
		}()
	}
	wg.Wait()
}

//func foo7() {
//	for {
//		nr, _ := http.NewRequest("POST", "https://bsc-mainnet.bk.nodereal.cc/v1/f34f62e7c0b343ef9f5bf80031a49cc2",
//			strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_getStorageAt\",\"params\":[\"0xbA2aE424d960c26247Dd6c32edC70B295c744C43\",\"0x0\",\"0x14da8d9\"]}"))
//		nr.Header.Add("Content-Type", "application/json")
//
//		client := &http.Client{Timeout: 60 * time.Second}
//		sTime := time.Now()
//		_, err := client.Do(nr)
//		eTime := time.Now()
//		duration := eTime.Sub(sTime).Milliseconds()
//		a := rand.Intn(100)
//		if a == 50 {
//			fmt.Println("******", duration, "******", err)
//		}
//	}
//}

func foo6() {
	nr, _ := http.NewRequest("POST", "https://bsc-mainnet.bk.nodereal.cc/v1/f34f62e7c0b343ef9f5bf80031a49cc2",
		strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_getStorageAt\",\"params\":[\"0xbA2aE424d960c26247Dd6c32edC70B295c744C43\",\"0x0\",\"0x14da8d9\"]}"))
	nr.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	sTime := time.Now()
	resp, err := client.Do(nr)
	eTime := time.Now()
	duration := eTime.Sub(sTime).Milliseconds()
	a := rand.Intn(100)
	if a == 50 {
		fmt.Println("^^^^^^^^", duration)
	}
	if err == nil {
		bodyByte, _ := ioutil.ReadAll(resp.Body)
		bodyStr := string(bodyByte)
		//fmt.Println(bodyStr)
		globalBoomer.RecordSuccess("http", "Test6", duration, int64(len(bodyStr)))
	} else {
		globalBoomer.RecordFailure("http", "Test6", duration, err.Error())
	}
}

var globalBoomer *boomer.Boomer

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	task1 := &boomer.Task{
		Name:   "foo1",
		Weight: 10,
		Fn:     foo1,
	}

	//task2 := &boomer.Task{
	//	Name:   "foo2",
	//	Weight: 10,
	//	Fn:     foo2,
	//}

	//task3 := &boomer.Task{
	//	Name:   "foo3",
	//	Weight: 20,
	//	Fn:     foo3,
	//}

	//task4 := &boomer.Task{
	//	Name:   "foo4",
	//	Weight: 10,
	//	Fn:     foo4,
	//}

	//task5 := &boomer.Task{
	//	Name:   "foo5",
	//	Weight: 10,
	//	Fn:     foo5,
	//}

	task6 := &boomer.Task{
		Name:   "foo6",
		Weight: 10,
		Fn:     foo6,
	}

	//foo5()

	numClients := 1000
	spawnRate := float64(1)
	globalBoomer = boomer.NewStandaloneBoomer(numClients, spawnRate)
	globalBoomer.AddOutput(boomer.NewConsoleOutputWithOptions(&boomer.OutputOptions{
		PercentTime: 90,
	}))
	//limiter := boomer.NewStableRateLimiter(, time.Second)
	//globalBoomer.SetRateLimiter(limiter)
	globalBoomer.OutputInterval = 8

	globalBoomer.Run(task1, task6)
}
