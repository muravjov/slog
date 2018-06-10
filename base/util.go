package base

import (
	"log"
	"math/rand"
	"os"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var seedDone = false

func RandStringBytes(n int) string {
	if !seedDone {
		seedDone = true
		rand.Seed(time.Now().UnixNano())
	}

	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}

func Assert(b bool) {
	if !b {
		panic("Assertion error")
	}
}

func CheckFatal(format string, err error) {
	if err != nil {
		log.Fatalf(format, err)
	}
}

func OpenLog(errFileName string) *os.File {
	logFile, err := os.OpenFile(errFileName,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.FileMode(0640))
	CheckFatal("Can't open: %s", err)
	return logFile
}
