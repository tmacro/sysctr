package main

import (
	"github.com/tmacro/sysctr/pkg/runner"
	"github.com/tmacro/sysctr/pkg/types"
)

type RmCmd struct {
	Spec string `short:"s" type:"existingfile" placeholder:"PATH" help:"Path to container specification." required:"true"`
}

func (r *RmCmd) Run(appCtx *AppContext) error {
	spec, err := types.ReadSpecFromFile(r.Spec)
	if err != nil {
		return err
	}

	err = runner.Remove(appCtx.Context, appCtx.Driver, spec, runner.RemoveOptions{})
	if err != nil {
		return err
	}

	return nil
}
