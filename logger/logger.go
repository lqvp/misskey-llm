package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Level int

const (
	LevelError Level = iota
	LevelInfo
	LevelDebug
)

var (
	currentLevel Level = LevelInfo
	InfoLogger   *log.Logger
	ErrorLogger  *log.Logger
	DebugLogger  *log.Logger
	fileWriter   io.Writer
	consoleWriter io.Writer
)

func Init(logDir string, level string) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("misskey-llm_%s.log", time.Now().Format("2006-01-02")))
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	consoleWriter = os.Stdout
	fileWriter = f

	SetLevel(level)

	return nil
}

func SetLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		currentLevel = LevelDebug
	case "error":
		currentLevel = LevelError
	default:
		currentLevel = LevelInfo
	}

	var consoleOut io.Writer
	var fileOut io.Writer

	switch currentLevel {
	case LevelDebug:
		consoleOut = consoleWriter
		fileOut = fileWriter
	case LevelInfo:
		consoleOut = consoleWriter
		fileOut = fileWriter
	case LevelError:
		consoleOut = consoleWriter
		fileOut = fileWriter
	}

	multiWriter := io.MultiWriter(consoleOut, fileOut)
	allWriter := io.MultiWriter(fileWriter)

	InfoLogger = log.New(multiWriter, "INFO  ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(multiWriter, "ERROR ", log.Ldate|log.Ltime|log.Lshortfile)
	DebugLogger = log.New(allWriter, "DEBUG ", log.Ldate|log.Ltime|log.Lshortfile)

	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	Info("Log level: %s", strings.ToUpper(level))
}

func Info(format string, v ...interface{}) {
	if currentLevel >= LevelInfo {
		InfoLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

func Error(format string, v ...interface{}) {
	ErrorLogger.Output(2, fmt.Sprintf(format, v...))
}

func Debug(format string, v ...interface{}) {
	if currentLevel >= LevelDebug {
		DebugLogger.Output(2, fmt.Sprintf(format, v...))
	}
}
