package main

import (
	"log"
	"os"
)

type logWriter struct {
	logFile *os.File
}

func (lw logWriter) Write(bytes []byte) (int, error) {
	return lw.logFile.Write(bytes)
}

func newLogger(path string) *logWriter {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	lw := new(logWriter)
	lw.logFile = f

	log.SetOutput(lw.logFile)
	return lw
}

func (lw *logWriter) Close() {
	lw.logFile.Close()
}
