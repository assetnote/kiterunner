package log

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
)

type LogFormat string

var (
	Pretty LogFormat = "pretty"
	JSON   LogFormat = "json"
	Text   LogFormat = "text"
)

var (
	stderr = zerolog.New(os.Stderr).With().Timestamp().Logger()

	// Logger for stdout specifically
	Stdout = zerolog.New(os.Stdout).With().Timestamp().Logger()

	globalFormat LogFormat = "pretty"

	Print  = stderr.Print
	Printf = stderr.Printf

	Fatal = stderr.Fatal
	Panic = stderr.Panic
	Error = stderr.Error
	Warn  = stderr.Warn
	Info  = stderr.Info
	Debug = stderr.Debug
	Trace = stderr.Trace
	Log   = stderr.Log

	Err       = stderr.Err
	With      = stderr.With
	WithLevel = stderr.WithLevel

	GetLevel = stderr.GetLevel
)

const (
	FatalLevel = zerolog.FatalLevel
	PanicLevel = zerolog.PanicLevel
	ErrorLevel = zerolog.ErrorLevel
	WarnLevel  = zerolog.WarnLevel
	InfoLevel  = zerolog.InfoLevel
	DebugLevel = zerolog.DebugLevel
	TraceLevel = zerolog.TraceLevel
)

func SetLevelString(level string) error {
	l, err := zerolog.ParseLevel(level)
	if err != nil {
		return err
	}

	stderr = stderr.Level(l)
	Stdout = Stdout.Level(l)
	return nil
}

var (
	ErrUnsupportedFormat = fmt.Errorf("unsupported format. supported 'json', 'pretty', 'text")
)

func GetLogFormat() LogFormat {
	return globalFormat
}

func SetFormat(format string) error {
	switch format {
	case "json", "":
		globalFormat = JSON
	case "pretty":
		stderr = stderr.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: false, TimeFormat: "\r3:04PM"})
		Stdout = Stdout.Output(zerolog.ConsoleWriter{Out: os.Stdout, NoColor: false, TimeFormat: "\r3:04PM"})
		globalFormat = Pretty
	case "text":
		stderr = stderr.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true, TimeFormat: "\r3:04PM"})
		Stdout = Stdout.Output(zerolog.ConsoleWriter{Out: os.Stdout, NoColor: true, TimeFormat: "\r3:04PM"})
		globalFormat = Text
	default:
		return ErrUnsupportedFormat
	}
	return nil
}
