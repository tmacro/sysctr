package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/docker/client"
	"github.com/rs/zerolog"
	"github.com/tmacro/sysctr/pkg/driver"
	"github.com/tmacro/sysctr/pkg/runner"
	"github.com/tmacro/sysctr/pkg/types"
)

type RunCmd struct {
	Spec      string `short:"s" type:"existingfile" placeholder:"PATH" help:"Path to container specification." required:"true"`
	NoCleanup bool   `short:"n" help:"Do not remove container after it exits." default:"false"`
}

func (r *RunCmd) Run(logger zerolog.Logger) error {
	spec, err := types.ReadSpecFromFile(r.Spec)
	if err != nil {
		return err
	}

	driver, err := driver.NewDockerDriver(client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	ctx := logger.WithContext(context.Background())
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	runOpts := runner.RunOptions{
		Cleanup: !r.NoCleanup,
	}

	exitCode, err := runner.Run(ctx, driver, spec, runOpts)
	if err != nil {
		return err
	}

	os.Exit(exitCode)

	// specHash, err := types.HashSpec(spec)
	// if err != nil {
	// 	return err
	// }

	// container, err := driver.FindExisting(ctx, spec)
	// if err != nil && err != ctr.ErrContainerNotFound {
	// 	return err
	// }

	// var needsRemoval bool = false
	// var containerID string

	// if container != nil {
	// 	needsRemoval = true
	// 	containerID = container.ID
	// 	if container.Status == ctr.Created {
	// 		needsRemoval = container.Hash != specHash
	// 	}
	// }

	// if needsRemoval {
	// 	err = driver.Remove(ctx, containerID)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	// if containerID == "" {
	// 	containerID, err = driver.Create(ctx, spec)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	// err = driver.Start(ctx, containerID)
	// if err != nil {
	// 	return err
	// }

	// if !r.NoCleanup {
	// 	err = driver.Remove(ctx, containerID)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	// err = driver.Run(ctx, spec, runOpts)
	// if err != nil {
	// 	return err
	// }

	// err := json.Unmarshal()

	// logger.Info().Msg("running container")

	// ctrd, err := client.New("/run/containerd/containerd.sock", client.WithDefaultNamespace("moby"))
	// if err != nil {
	// 	return err
	// }

	// defer ctrd.Close()

	// ctx := context.Background()

	// containers, err := ctrd.Containers(ctx, "id==98636d3abb40f926746d5a3650b4c8e9475d233abb338defb2eb9c17e454584f")
	// if err != nil {
	// 	return err
	// }

	// spew.Dump(containers)

	// _, err = ctrd.LoadContainer(ctx, config.Name)
	// if err != nil {
	// 	return err
	// }

	return nil
}
