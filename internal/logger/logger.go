package logger

import (
	"log"
	"os"
)

type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

type StandardLogger struct {
	debug *log.Logger
	info  *log.Logger
	warn  *log.Logger
	error *log.Logger
	fatal *log.Logger
}

func New(level string, format string) Logger {
	var flags int
	
	if format == "json" {
		flags = log.LstdFlags | log.Lmsgprefix
	} else {
		flags = log.LstdFlags
	}

	return &StandardLogger{
		debug: log.New(os.Stdout, "DEBUG: ", flags),
		info:  log.New(os.Stdout, "INFO: ", flags),
		warn:  log.New(os.Stdout, "WARN: ", flags),
		error: log.New(os.Stderr, "ERROR: ", flags),
		fatal: log.New(os.Stderr, "FATAL: ", flags),
	}
}

func (l *StandardLogger) Debug(args ...interface{}) {
	l.debug.Println(args...)
}

func (l *StandardLogger) Info(args ...interface{}) {
	l.info.Println(args...)
}

func (l *StandardLogger) Warn(args ...interface{}) {
	l.warn.Println(args...)
}

func (l *StandardLogger) Error(args ...interface{}) {
	l.error.Println(args...)
}

func (l *StandardLogger) Fatal(args ...interface{}) {
	l.fatal.Println(args...)
	os.Exit(1)
}

func (l *StandardLogger) Debugf(format string, args ...interface{}) {
	l.debug.Printf(format, args...)
}

func (l *StandardLogger) Infof(format string, args ...interface{}) {
	l.info.Printf(format, args...)
}

func (l *StandardLogger) Warnf(format string, args ...interface{}) {
	l.warn.Printf(format, args...)
}

func (l *StandardLogger) Errorf(format string, args ...interface{}) {
	l.error.Printf(format, args...)
}

func (l *StandardLogger) Fatalf(format string, args ...interface{}) {
	l.fatal.Fatalf(format, args...)
}

var Default Logger

func Init(level string, format string) {
	Default = New(level, format)
}

func Debug(args ...interface{}) {
	Default.Debug(args...)
}

func Info(args ...interface{}) {
	Default.Info(args...)
}

func Warn(args ...interface{}) {
	Default.Warn(args...)
}

func Error(args ...interface{}) {
	Default.Error(args...)
}

func Fatal(args ...interface{}) {
	Default.Fatal(args...)
}

func Debugf(format string, args ...interface{}) {
	Default.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	Default.Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	Default.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	Default.Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	Default.Fatalf(format, args...)
}