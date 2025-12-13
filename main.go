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

// TODO: check the search in bigger ones not all is highlighted???
// PERF:split it up to multiple go routines
func SearchLastNLines(fileName string, readLineCount int16, search string) (string, error) {
	searchTerm := []byte(search)
	f, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer f.Close()

	const chunkSize = 1024
	// const chunkSize = 8096
	// const chunkSize = 16192
	// const chunkSize = 32384
	stat, _ := f.Stat()
	size := stat.Size()

	var buf []byte
	var readSize int64
	var newlineCnt int16

	for size > 0 && newlineCnt <= readLineCount {
		readSize = min(size, chunkSize)
		size -= readSize

		chunk := make([]byte, readSize)
		_, err := f.ReadAt(chunk, size)
		if err != nil {
			return "", err
		}

		if len(searchTerm) != 0 && bytes.Contains(chunk, searchTerm) {
			highlightedSearch := append(append(highlightColor, searchTerm...), resetColor...)
			highlighted := bytes.ReplaceAll(
				chunk,
				searchTerm,
				highlightedSearch,
			)
			chunk = highlighted
		}

		// print only as much lines as given readLineCount-> -n flag
		for i := len(chunk) - 1; i >= 0; i-- {
			if chunk[i] == '\n' {
				newlineCnt++
				if newlineCnt > readLineCount {
					buf = append(buf, chunk[i+1:]...)
					return string(buf), nil
				}
			}
		}

		buf = append(buf, chunk...)
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
