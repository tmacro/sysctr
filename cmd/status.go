package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/tmacro/sysctr/pkg/driver"
	"github.com/tmacro/sysctr/pkg/runner"
	"github.com/tmacro/sysctr/pkg/types"
)

type StatusCmd struct {
	Spec string `short:"s" type:"existingfile" placeholder:"PATH" help:"Path to container specification." required:"true"`
}

func (p *StatusCmd) Run(logger zerolog.Logger) error {
	spec, err := types.ReadSpecFromFile(p.Spec)
	if err != nil {
		return err
	}

	driver, err := driver.NewDockerDriver()
	if err != nil {
		return err
	}

	ctx := context.Background()

	ctx = logger.WithContext(ctx)

	status, err := runner.Status(ctx, driver, spec)
	if err != nil {
		return err
	}

	statusJson, err := json.Marshal(status)
	if err != nil {
		return err
	}

	fmt.Println(string(statusJson))

	return nil
}
