package main

import (
	"bufio"
	"log"
	"os"
	"strings"
)

func readConfFile(filePath string, a adder) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Open file %s error: %v", filePath, err)
	}
	defer file.Close()
	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil {
		log.Printf("Read file %s error: %v", filePath, err)
	}
	parselines(string(buffer[:n]), a)
}

func parselines(str string, a adder) {
	scanner := bufio.NewScanner(strings.NewReader(string(str)))
	for scanner.Scan() {
		if scanner.Bytes()[0] != byte('#') {
			line := scanner.Text()
			i := strings.Index(line, ":")
			a.add(strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]))
		}
	}
}
