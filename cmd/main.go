package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"

	"github.com/tmacro/sysctr/pkg/driver"
	_ "github.com/tmacro/sysctr/pkg/driver/containerd"
	_ "github.com/tmacro/sysctr/pkg/driver/docker"
)

var CLI struct {
	LogLevel  string    `help:"Set the log level." enum:"trace,debug,info,warn,error" default:"debug"`
	LogFormat string    `enum:"json,text" default:"text" help:"Set the log format. (json, text)"`
	Config    string    `short:"c" help:"Path to sysctr configuration." type:"existingfile" placeholder:"PATH"`
	Pull      PullCmd   `cmd:"" help:"Pull a container's image."`
	Run       RunCmd    `cmd:"" help:"Run a container."`
	Status    StatusCmd `cmd:"" help:"Get the status of a container."`
	Stop      StopCmd   `cmd:"" help:"Stop a container."`
	Rm        RmCmd     `cmd:"" help:"Remove a container."`
}

type AppContext struct {
	Logger  zerolog.Logger
	Driver  driver.Driver
	Context context.Context
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

	ctx := context.Background()

	ctx = logger.WithContext(ctx)

	drv, err := loadDriver(ctx, CLI.Config, "")
	if err != nil {
		logger.Fatal().Err(err).Msg("error loading driver")
	}

	appCtx := AppContext{
		Logger:  logger,
		Driver:  drv,
		Context: ctx,
	}

	err = cmd.Run(&appCtx)
	if err != nil {
		logger.Fatal().Err(err).Msg("error running command")
	}
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

type sysctrConfig struct {
	Driver map[string]json.RawMessage `json:"driver"`
}

func loadDriver(ctx context.Context, configPath string, driverID string) (driver.Driver, error) {
	var config sysctrConfig
	if configPath != "" {
		file, err := os.Open(configPath)
		if err != nil {
			return nil, err
		}

		defer file.Close()

		err = json.NewDecoder(file).Decode(&config)
		if err != nil {
			return nil, err
		}
	}

	availableDriverConfigs := []string{}
	if config.Driver != nil {
		for k := range config.Driver {
			availableDriverConfigs = append(availableDriverConfigs, k)
		}
	}

	if driverID == "" && len(availableDriverConfigs) == 1 {
		driverID = availableDriverConfigs[0]
	} else if driverID == "" && len(availableDriverConfigs) > 1 {
		return nil, errors.New("multiple drivers configured, must specify driver ID")
	}

	driverConf := []byte{}
	if config.Driver != nil {
		dc, ok := config.Driver[driverID]
		if ok {
			driverConf = dc
		}
	}

	return driver.LoadDriver(ctx, driverID, driverConf)
}
