package onerror

import "log"

func Log(err error) {
	Logf("", err)
}

func Logf(msg string, err error) {
	if err != nil {
		log.Fatalf("\n%s%s", msg, err)
	}
}
