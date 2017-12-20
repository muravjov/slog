package util

import (
	"time"
	"math/rand"
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
