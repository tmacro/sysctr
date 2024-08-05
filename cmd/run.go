package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/tmacro/sysctr/pkg/runner"
	"github.com/tmacro/sysctr/pkg/types"
)

type RunCmd struct {
	Spec      string `short:"s" type:"existingfile" placeholder:"PATH" help:"Path to container specification." required:"true"`
	NoCleanup bool   `short:"n" help:"Do not remove container after it exits." default:"false"`
}

func (r *RunCmd) Run(appCtx *AppContext) error {
	spec, err := types.ReadSpecFromFile(r.Spec)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(appCtx.Context, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	runOpts := runner.RunOptions{
		Cleanup: !r.NoCleanup,
	}

	exitCode, err := runner.Run(ctx, appCtx.Driver, spec, runOpts)
	if err != nil {
		return err
	}

	os.Exit(exitCode)

	return nil
}
