package main

import (
	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

func main() {
	if err := sentry.Init(sentry.ClientOptions{Dsn: ""}); err != nil {
		panic(err)
	}
	logger := newLogger()
	if err := fun1(); err != nil {
		logger.Errorw("error!", "error", err)
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

type SentryZapCore struct {
	enabledLevel zapcore.Level
	fields map[string]interface{}
}

func (s *SentryZapCore) Enabled(level zapcore.Level) bool {
	return s.enabledLevel <= level
}

func (s *SentryZapCore) With(fields []zapcore.Field) zapcore.Core {
	copied := make(map[string]interface{}, len(s.fields))
	for k, v := range s.fields {
		copied[k] = v
	}
	encoder := zapcore.NewMapObjectEncoder()
	for _, f := range fields {
		f.AddTo(encoder)
	}
	for k, v := range encoder.Fields {
		copied[k] = v
	}
	return &SentryZapCore{fields: copied, enabledLevel: s.enabledLevel}
}

func (s *SentryZapCore) Check(entry zapcore.Entry, checkedEntry *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if s.Enabled(entry.Level) {
		checkedEntry.AddCore(entry, s)
	}
	return checkedEntry
}

func (s *SentryZapCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	event := sentry.NewEvent()
	event.Message = entry.Message
	event.Timestamp = entry.Time.Unix()
	event.Level = zapLevelToSentryLevel(entry.Level)
	event.Platform = "Golang"
	exceptions := make([]sentry.Exception, 0)
	for _, f := range fields {
		if f.Type == zapcore.ErrorType {
			err := f.Interface.(error)
			trace := sentry.ExtractStacktrace(err)
			if trace == nil {
				trace = sentry.NewStacktrace()
			}
			exceptions = append(exceptions, sentry.Exception{
				Type:       entry.Message,
				Value:      entry.Caller.TrimmedPath(),
				Stacktrace: trace,
			})
		}
	}
	event.Exception = exceptions
	sentry.CaptureEvent(event)
	defer s.Sync()
	return nil
}

func (s *SentryZapCore) Sync() error {
	sentry.Flush(2 * time.Second)
	return nil
}

func newLogger() *zap.SugaredLogger {
	config := zap.NewProductionConfig()
	logger, _ := config.Build()
	logger = logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, &SentryZapCore{enabledLevel: zapcore.DebugLevel})
	}))
	return logger.Sugar()
}

func zapLevelToSentryLevel(level zapcore.Level) sentry.Level {
	switch level {
	case zapcore.DebugLevel:
		return sentry.LevelDebug
	case zapcore.InfoLevel:
		return sentry.LevelInfo
	case zapcore.WarnLevel:
		return sentry.LevelWarning
	case zapcore.ErrorLevel:
		return sentry.LevelError
	case zapcore.DPanicLevel:
		return sentry.LevelFatal
	case zapcore.PanicLevel:
		return sentry.LevelFatal
	case zapcore.FatalLevel:
		return sentry.LevelFatal
	default:
		return sentry.LevelFatal
	}
}
