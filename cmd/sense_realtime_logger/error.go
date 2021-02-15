package main

import (
	"log"
	"time"

	"golang.org/x/time/rate"
)

type errorMsg struct {
	component string
	err       error
}

// Rate limited error logging
func errorLoggerLoop(errCh chan errorMsg) {
	limiters := make(map[string]*rate.Limiter)
	for errMsg := range errCh {
		limiter, ok := limiters[errMsg.component]
		if !ok {
			limiter = rate.NewLimiter(rate.Every(30*time.Second), 10)
			limiters[errMsg.component] = limiter
		}

		if limiter.Allow() {
			log.Printf("%s: %s", errMsg.component, errMsg.err.Error())
		}
	}
}

func logErrorMsg(component string, err error, errCh chan errorMsg) {
	errMsg := errorMsg{component, err}
	if errCh != nil {
		select {
		case errCh <- errMsg:
		default:
			// Noop
		}
	}
}
