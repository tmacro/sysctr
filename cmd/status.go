package main

import (
	"encoding/json"
	"fmt"

	"github.com/tmacro/sysctr/pkg/runner"
	"github.com/tmacro/sysctr/pkg/types"
)

type StatusCmd struct {
	Spec string `short:"s" type:"existingfile" placeholder:"PATH" help:"Path to container specification." required:"true"`
}

func (p *StatusCmd) Run(appCtx *AppContext) error {
	spec, err := types.ReadSpecFromFile(p.Spec)
	if err != nil {
		return err
	}

	status, err := runner.Status(appCtx.Context, appCtx.Driver, spec)
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
