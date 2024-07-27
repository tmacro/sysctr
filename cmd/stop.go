package main

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/tmacro/sysctr/pkg/driver"
	"github.com/tmacro/sysctr/pkg/runner"
	"github.com/tmacro/sysctr/pkg/types"
)

type StopCmd struct {
	Spec string `short:"s" type:"existingfile" placeholder:"PATH" help:"Path to container specification." required:"true"`
}

func (s *StopCmd) Run(logger zerolog.Logger) error {
	spec, err := types.ReadSpecFromFile(s.Spec)
	if err != nil {
		return err
	}

	driver, err := driver.NewDockerDriver()
	if err != nil {
		return err
	}

	ctx := context.Background()

	ctx = logger.WithContext(ctx)

	err = runner.Stop(ctx, driver, spec, runner.StopOptions{})
	if err != nil {
		return err
	}

	return nil
}
