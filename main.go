package main

import (
	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"strconv"
	"zap-sentry.example.local/log"
)

func main() {
	if err := sentry.Init(sentry.ClientOptions{Dsn: ""}); err != nil {
		panic(err)
	}
	logger := log.NewLogger()
	if err := fun1(); err != nil {
		logger.Errorw("error!", "error", err)
	}
	if _, err := strconv.Atoi("A"); err != nil {
		logger.Errorw(err.Error(), "error", err)
	}
}

func fun1() error {
	return fun2()
}

func fun2() error {
	return fun3()
}

func fun3() error {
	return errors.New("ERROR!")
}
