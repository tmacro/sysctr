package runner

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/tmacro/sysctr/pkg/driver"
	"github.com/tmacro/sysctr/pkg/types"
)

type RemoveOptions struct {
	Force   bool
	Timeout int
}

func Remove(ctx context.Context, drv driver.Driver, spec *types.Spec, opts RemoveOptions) error {
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

	return drv.RemoveContainer(ctx, status.ID)
}
