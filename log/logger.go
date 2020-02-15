package log

import (
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strings"
	"time"
)

type SentryZapCore struct {
	enabledLevel zapcore.Level
	fields       map[string]interface{}
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
				trace = newStackTrace()
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

func NewLogger() *zap.SugaredLogger {
	config := zap.NewProductionConfig()
	logger, _ := config.Build()
	logger = logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, &SentryZapCore{enabledLevel: zapcore.DebugLevel})
	}))
	return logger.Sugar()
}

func newStackTrace() *sentry.Stacktrace {
	trace := sentry.NewStacktrace()
	filteredFrames := make([]sentry.Frame, 0)

	for _, frame := range trace.Frames {
		if frame.Module == "runtime" || frame.Module == "testing" {
			continue
		}
		if (strings.HasPrefix(frame.Module, "zap-sentry.example.local/log") || strings.HasPrefix(frame.Function, "go.uber.org/zap")) &&
			!strings.HasSuffix(frame.Module, "_test") {
			continue
		}
		filteredFrames = append(filteredFrames, frame)
	}

	return &sentry.Stacktrace{Frames: filteredFrames}
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
