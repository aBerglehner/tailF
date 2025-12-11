package main

import (
	"flag"
	"fmt"
	"os"
)

const MaxReadLineCount int16 = 32000

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run main.go <name>")
		return
	}
	fileName := os.Args[len(os.Args)-1]
	n := flag.Int("n", 10, "Number of lines")
	flag.Parse()

	fmt.Printf("fileName: %v\n", fileName)
	fmt.Printf("n: %v\n", *n)
	fmt.Println("")

	readLineCount := int16(min(*n, int(MaxReadLineCount)))
	str := run(fileName, readLineCount)
	fmt.Println(str)
}

func run(fileName string, readLineCount int16) string {
	str, err := ReadLastNLines(fileName, readLineCount)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return err.Error()
	}
	return str
}

func ReadLastNLines(fileName string, readLineCount int16) (string, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer f.Close()

	const chunkSize = 8096
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
