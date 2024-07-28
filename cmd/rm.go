package main

import (
	"context"

	"github.com/docker/docker/client"
	"github.com/rs/zerolog"
	"github.com/tmacro/sysctr/pkg/driver"
	"github.com/tmacro/sysctr/pkg/runner"
	"github.com/tmacro/sysctr/pkg/types"
)

type RmCmd struct {
	Spec string `short:"s" type:"existingfile" placeholder:"PATH" help:"Path to container specification." required:"true"`
}

func (r *RmCmd) Run(logger zerolog.Logger) error {
	spec, err := types.ReadSpecFromFile(r.Spec)
	if err != nil {
		return err
	}

	driver, err := driver.NewDockerDriver(client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	ctx := context.Background()

	ctx = logger.WithContext(ctx)

	err = runner.Remove(ctx, driver, spec, runner.RemoveOptions{})
	if err != nil {
		return err
	}

	return nil
}
