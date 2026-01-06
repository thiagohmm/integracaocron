package logger

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/thiagohmm/integracaocron/pkg/tracing"
)

type Logger struct {
	*log.Logger
}

func New() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile),
	}
}

func (l *Logger) logWithTrace(ctx context.Context, level, message string, args ...interface{}) {
	traceID := tracing.GetTraceID(ctx)
	spanID := tracing.GetSpanID(ctx)
	
	prefix := fmt.Sprintf("[%s]", level)
	if traceID != "" {
		prefix = fmt.Sprintf("[%s][trace:%s][span:%s]", level, traceID[:8], spanID[:8])
	}
	
	fullMessage := fmt.Sprintf(message, args...)
	l.Logger.Printf("%s %s", prefix, fullMessage)
}

func (l *Logger) Info(ctx context.Context, message string, args ...interface{}) {
	l.logWithTrace(ctx, "INFO", message, args...)
	tracing.AddEvent(ctx, "log.info", tracing.StringAttr("message", fmt.Sprintf(message, args...)))
}

func (l *Logger) Error(ctx context.Context, message string, args ...interface{}) {
	l.logWithTrace(ctx, "ERROR", message, args...)
	tracing.AddEvent(ctx, "log.error", tracing.StringAttr("message", fmt.Sprintf(message, args...)))
}

func (l *Logger) Warn(ctx context.Context, message string, args ...interface{}) {
	l.logWithTrace(ctx, "WARN", message, args...)
	tracing.AddEvent(ctx, "log.warn", tracing.StringAttr("message", fmt.Sprintf(message, args...)))
}

func (l *Logger) Debug(ctx context.Context, message string, args ...interface{}) {
	l.logWithTrace(ctx, "DEBUG", message, args...)
	tracing.AddEvent(ctx, "log.debug", tracing.StringAttr("message", fmt.Sprintf(message, args...)))
}

var DefaultLogger = New()

// Global functions for convenience
func Info(ctx context.Context, message string, args ...interface{}) {
	DefaultLogger.Info(ctx, message, args...)
}

func Error(ctx context.Context, message string, args ...interface{}) {
	DefaultLogger.Error(ctx, message, args...)
}

func Warn(ctx context.Context, message string, args ...interface{}) {
	DefaultLogger.Warn(ctx, message, args...)
}

func Debug(ctx context.Context, message string, args ...interface{}) {
	DefaultLogger.Debug(ctx, message, args...)
}