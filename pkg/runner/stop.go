package runner

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/tmacro/sysctr/pkg/driver"
	"github.com/tmacro/sysctr/pkg/types"
)

type StopOptions struct {
	Timeout int
}

func Stop(ctx context.Context, drv driver.Driver, spec *types.Spec, opts StopOptions) error {
	logger := zerolog.Ctx(ctx)

	status, err := drv.FindContainer(ctx, spec.Name, map[string]string{
		LabelSysCtr: "true",
		LabelName:   spec.Name,
	})

	if err == driver.ErrContainerNotFound {
		logger.Info().Msg("container not found")
		os.Exit(1)
	}

	if err != nil {
		return err
	}

	return drv.StopContainer(ctx, status.ID)
}
