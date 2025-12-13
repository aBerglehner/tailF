package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"syscall"
	"time"
)

const MaxReadLineCount int16 = 32000

var (
	highlightColor = []byte("\033[31m")
	resetColor     = []byte("\033[0m")
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("file argument missing! sample: main.go test.txt")
		return
	}
	fileName := os.Args[len(os.Args)-1]
	n := flag.Int("n", 100, "Number of lines")
	flag.Parse()

	fmt.Printf("fileName: %v\n", fileName)
	fmt.Printf("n: %v\n", *n)
	fmt.Println("")

	readLineCount := int16(min(*n, int(MaxReadLineCount)))

	initSearchCh := make(chan string)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter search term (empty to clear): ")
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
	str, err := SearchLastNLines(fileName, readLineCount, searchTerm)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return err.Error()
	}
	return str
}

// PERF:split it up to multiple go routines
func SearchLastNLines(fileName string, readLineCount int16, search string) (string, error) {
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

	data, err := syscall.Mmap(
		int(f.Fd()),
		0,
		int(size),
		syscall.PROT_READ,
		syscall.MAP_SHARED,
	)
	if err != nil {
		return "", err
	}
	defer syscall.Munmap(data)

	var buf []byte
	var newlineCnt int16

	highlightedSearch := append(
		append([]byte{}, highlightColor...),
		append(searchTerm, resetColor...)...,
	)

	// walk backwards in chunks (logical chunks, no copying)
	// const chunkSize = 1024
	// const chunkSize = 4096
	// const chunkSize = 8192
	// const chunkSize = 16384
	const chunkSize = 20480
	// const chunkSize = 24576

	for end := len(data); end > 0 && newlineCnt <= readLineCount; {
		start := max(end-chunkSize, 0)

		chunk := data[start:end] // ZERO COPY

		// optional highlight (this allocates only if match exists)
		if len(searchTerm) != 0 && bytes.Contains(chunk, searchTerm) {
			chunk = bytes.ReplaceAll(
				chunk,
				searchTerm,
				highlightedSearch,
			)
		}

		// will use bytes.Count as it will use simd
		newlineCnt += int16(bytes.Count(chunk, []byte{'\n'}))
		if newlineCnt > readLineCount {
			// buf = append(buf, chunk[i+1:]...)
			buf = append(buf, chunk...)
			return string(buf), nil
		}

		// print only as much lines as given readLineCount-> -n flag
		// for i := len(chunk) - 1; i >= 0; i-- {
		// 	if chunk[i] == '\n' {
		// 		newlineCnt++
		// 		if newlineCnt > readLineCount {
		// 			buf = append(buf, chunk[i+1:]...)
		// 			return string(buf), nil
		// 		}
		// 	}
		// }

		buf = append(buf, chunk...)
		end = start
	}

	return string(buf), nil
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
