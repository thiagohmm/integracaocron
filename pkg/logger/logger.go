package logger

import (
	"context"
	"fmt"
	"log"
	"os"

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
	formattedMessage := fmt.Sprintf(message, args...)
	tracing.AddEvent(ctx, "log.info", tracing.StringAttr("message", formattedMessage))
	tracing.LogInfo(ctx, formattedMessage)
}

func (l *Logger) Error(ctx context.Context, message string, args ...interface{}) {
	l.logWithTrace(ctx, "ERROR", message, args...)
	formattedMessage := fmt.Sprintf(message, args...)
	tracing.AddEvent(ctx, "log.error", tracing.StringAttr("message", formattedMessage))
	tracing.LogError(ctx, formattedMessage)
}

func (l *Logger) Warn(ctx context.Context, message string, args ...interface{}) {
	l.logWithTrace(ctx, "WARN", message, args...)
	formattedMessage := fmt.Sprintf(message, args...)
	tracing.AddEvent(ctx, "log.warn", tracing.StringAttr("message", formattedMessage))
	tracing.LogWarn(ctx, formattedMessage)
}

func (l *Logger) Debug(ctx context.Context, message string, args ...interface{}) {
	l.logWithTrace(ctx, "DEBUG", message, args...)
	formattedMessage := fmt.Sprintf(message, args...)
	tracing.AddEvent(ctx, "log.debug", tracing.StringAttr("message", formattedMessage))
	tracing.LogDebug(ctx, formattedMessage)
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