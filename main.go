package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	MaxReadLineCount int16 = 32000
	DefaultLineCount int   = 600
)

var (
	highlightColor = []byte("\033[31m")
	resetColor     = []byte("\033[0m")
)

// ❯ sudo perf stat go run main.go big.txt
func main() {
	if len(os.Args) < 2 {
		fmt.Println("file argument missing! sample: main.go test.txt")
		return
	}
	fileName := os.Args[len(os.Args)-1]
	n := flag.Int("n", DefaultLineCount, "Number of lines")
	flag.Parse()

	readLineCount := int16(min(*n, int(MaxReadLineCount)))

	initSearchCh := make(chan string)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		initSearchCh <- ""
		for {
			text, _ := reader.ReadString('\n')
			text = strings.TrimSpace(text)
			initSearchCh <- text
		}
	}()

	f, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	tailCh := make(chan string)
	go tailF(f, fileName, tailCh)

	for {
		select {
		case s := <-tailCh:
			fmt.Printf("%s", s)
		case searchTerm := <-initSearchCh:
			// \033[H  -> moves the cursor to top-left
			// \033[2J -> clears the screen
			fmt.Print("\033[H\033[2J")
			str := run(fileName, readLineCount, searchTerm)
			fmt.Printf("%s", str)
		}
	}
}

func run(fileName string, readLineCount int16, searchTerm string) string {
	str, err := ParallelSearchLastNLines(fileName, readLineCount, searchTerm)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return err.Error()
	}
	return str
}

func ParallelSearchLastNLines(fileName string, readLineCount int16, search string) (string, error) {
	searchTerm := []byte(search)

	f, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", err
	}
	size := stat.Size()
	if size == 0 {
		return "", nil
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return "", err
	}
	defer syscall.Munmap(data)

	highlightedSearch := append(
		append([]byte{}, highlightColor...),
		append(searchTerm, resetColor...)...,
	)

	offset := findOffset(data, readLineCount)
	relevantData := data[offset:]

	workers := runtime.NumCPU()
	chunks := splitTasks(relevantData, workers)

	var wg sync.WaitGroup
	for i := range chunks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if len(searchTerm) != 0 && bytes.Contains(chunks[i], searchTerm) {
				chunks[i] = bytes.ReplaceAll(
					chunks[i],
					searchTerm,
					highlightedSearch,
				)
			}
		}(i)
	}
	wg.Wait()

	return string(bytes.Join(chunks, nil)), nil
}

func splitTasks[T any](tasks []T, n int) [][]T {
	var chunks [][]T
	length := len(tasks)
	if n <= 0 {
		n = 1
	}
	for i := 0; i < n; i++ {
		start := i * length / n
		end := (i + 1) * length / n
		chunks = append(chunks, tasks[start:end])
	}
	return chunks
}

// this is not much faster than a primitive one but it could also scale better
// with newer cpus cause of better use of ipc
// PERF: try stuff with Count as it uses bytecode simd
// newlineCnt += int16(bytes.Count(chunk, []byte{'\n'}))
func findOffset(data []byte, readLineCount int16) int {
	var newlineCount int16 = 0
	for i := len(data) - 1; i >= 4; i -= 5 {
		if data[i] == '\n' {
			newlineCount++
			if newlineCount == readLineCount {
				return i + 1
			}
		}
		if data[i-1] == '\n' {
			newlineCount++
			if newlineCount == readLineCount {
				return i
			}
		}
		if data[i-2] == '\n' {
			newlineCount++
			if newlineCount == readLineCount {
				return i - 1
			}
		}
		if data[i-3] == '\n' {
			newlineCount++
			if newlineCount == readLineCount {
				return i - 2
			}
		}
		if data[i-4] == '\n' {
			newlineCount++
			if newlineCount == readLineCount {
				return i - 3
			}
		}
	}

	for i := min(4, len(data)-1); i >= 0; i-- {
		if data[i] == '\n' {
			newlineCount++
			if newlineCount == readLineCount {
				return i + 1
			}
		}
	}
	return 0
}

func tailF(f *os.File, path string, tailCh chan<- string) {
	offset, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		log.Fatal(err)
	}
	reader := bufio.NewReader(f)
	for {
		// Try reading a line
		line, err := reader.ReadString('\n')
		if err == nil {
			tailCh <- line
			offset += int64(len(line))
			continue
		}

		// If nothing new: sleep and check again
		// TODO: try with lock
		time.Sleep(3000 * time.Millisecond)

		// Check if file grew
		stat, err := os.Stat(path)
		if err != nil {
			log.Fatal(err)
		}

		// // If file was truncated (e.g., rotated), reset
		if stat.Size() < offset {
			f.Close()
			f, _ = os.Open(path)
			reader = bufio.NewReader(f)
			offset = 0
			continue
		}
	}
}
