package main

import (
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// сюда писать код

func runJob(j job, wg *sync.WaitGroup, in, out chan interface{}) {
	defer runtime.Gosched()
	defer wg.Done()
	defer close(out)
	j(in, out)
}

// ExecutePipeline ...
func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	in := make(chan interface{})

	for _, job := range jobs {
		wg.Add(1)
		out := make(chan interface{})
		go runJob(job, wg, in, out)
		in = out
	}

	wg.Wait()
}

func handleSingleleHash(data string, out chan interface{}, mu *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()

	wgSync := &sync.WaitGroup{}
	signChanCrc32 := make(chan string, 1)
	signChanCrc32Md5 := make(chan string, 1)

	wgSync.Add(2)

	go func(wg *sync.WaitGroup, data string, out chan string) {
		defer wg.Done()
		defer close(out)

		mu.Lock()
		md5 := DataSignerMd5(data)
		mu.Unlock()
		out <- DataSignerCrc32(md5)
	}(wgSync, data, signChanCrc32Md5)

	go func(wg *sync.WaitGroup, data string, out chan string) {
		defer wg.Done()
		defer close(out)

		out <- DataSignerCrc32(data)
	}(wgSync, data, signChanCrc32)

	wgSync.Wait()

	left := <-signChanCrc32
	right := <-signChanCrc32Md5

	out <- left + "~" + right
}

func handleMultiHash(data string, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()

	wgSync := &sync.WaitGroup{}
	results := make([]string, 6)

	for idx := range results {
		wgSync.Add(1)

		go func(data string, idx int, results []string, wg *sync.WaitGroup) {
			defer wg.Done()
			hash := DataSignerCrc32(strconv.Itoa(idx) + data)
			results[idx] = hash
		}(data, idx, results, wgSync)
	}

	wgSync.Wait()

	out <- strings.Join(results, "")
}

// SingleHash ...
func SingleHash(in, out chan interface{}) {
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	for packet := range in {
		wg.Add(1)
		go handleSingleleHash(fmt.Sprintf("%d", packet), out, mu, wg)
	}

	wg.Wait()
}

// MultiHash ...
func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}

	for packet := range in {
		wg.Add(1)
		go handleMultiHash(fmt.Sprintf("%s", packet), out, wg)
	}

	wg.Wait()
}

// CombineResults ...
func CombineResults(in, out chan interface{}) {
	results := make([]string, 0)

	for packet := range in {
		results = append(results, fmt.Sprintf("%s", packet))
	}

	sort.Strings(results)

	out <- strings.Join(results, "_")
}
