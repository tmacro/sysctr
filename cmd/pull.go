package main

import (
	"github.com/tmacro/sysctr/pkg/types"
)

type PullCmd struct {
	Spec string `short:"s" type:"existingfile" placeholder:"PATH" help:"Path to container specification." required:"true"`
}

func (p *PullCmd) Run(appCtx *AppContext) error {
	spec, err := types.ReadSpecFromFile(p.Spec)
	if err != nil {
		return err
	}

	err = appCtx.Driver.PullImage(appCtx.Context, spec.Image)
	if err != nil {
		return err
	}

	return nil
}
