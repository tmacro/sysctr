package runner

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/tmacro/sysctr/pkg/driver"
	"github.com/tmacro/sysctr/pkg/types"
)

func Status(ctx context.Context, drv driver.Driver, spec *types.Spec) (types.ContainerState, error) {
	logger := zerolog.Ctx(ctx)

	container, err := drv.FindContainer(ctx, spec.Name, map[string]string{
		LabelSysCtr: "true",
		LabelName:   spec.Name,
	})

	if err == driver.ErrContainerNotFound {
		logger.Info().Msg("container not found")
		os.Exit(1)
	}

	if err != nil {
		return types.ContainerState{}, err
	}

	hashLabel := container.Labels[LabelSpecHash]

	status := types.ContainerState{
		Id:         container.ID,
		Status:     container.Status.String(),
		ConfigHash: hashLabel,
	}

	if container.Status == driver.Stopped {
		status.ExitCode = &container.ExitCode
	}

	return status, nil
}
