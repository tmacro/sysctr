package main

import (
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
)

var CLI struct {
	LogLevel  string  `help:"Set the log level." enum:"trace,debug,info,warn,error" default:"debug"`
	LogFormat string  `enum:"json,text" default:"text" help:"Set the log format. (json, text)"`
	Config    string  `short:"c" help:"Path to sysctr configuration." type:"existingfile" placeholder:"PATH"`
	Run       RunCmd  `cmd:"" help:"Run a container."`
	Pull      PullCmd `cmd:"" help:"Pull a container's image."`
	Stop      StopCmd `cmd:"" help:"Stop a container."`
	Rm        RmCmd   `cmd:"" help:"Remove a container."`
}

func setupLogger(level, format string) zerolog.Logger {
	var lvl zerolog.Level
	switch level {
	case "trace":
		lvl = zerolog.TraceLevel
	case "debug":
		lvl = zerolog.DebugLevel
	case "info":
		lvl = zerolog.InfoLevel
	case "warn":
		lvl = zerolog.WarnLevel
	case "error":
		lvl = zerolog.ErrorLevel
	}

	var writer io.Writer
	switch format {
	case "json":
		writer = os.Stdout
	case "text":
		writer = zerolog.ConsoleWriter{Out: os.Stdout}
	}
	return zerolog.New(writer).Level(lvl).With().Timestamp().Logger()
}

func main() {
	cmd := kong.Parse(&CLI,
		kong.Name("sysctr"),
		kong.Description("Simple container runner."),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)

	logger := setupLogger(CLI.LogLevel, CLI.LogFormat)

	err := cmd.Run(logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("error running command")
	}
}
