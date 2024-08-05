package main

import (
	"github.com/tmacro/sysctr/pkg/runner"
	"github.com/tmacro/sysctr/pkg/types"
)

type StopCmd struct {
	Spec string `short:"s" type:"existingfile" placeholder:"PATH" help:"Path to container specification." required:"true"`
}

func (s *StopCmd) Run(appCtx *AppContext) error {
	spec, err := types.ReadSpecFromFile(s.Spec)
	if err != nil {
		return err
	}

	err = runner.Stop(appCtx.Context, appCtx.Driver, spec, runner.StopOptions{})
	if err != nil {
		return err
	}

	return nil
}
