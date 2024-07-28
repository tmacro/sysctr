package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/rs/zerolog"
	"github.com/tmacro/sysctr/pkg/driver"
	"github.com/tmacro/sysctr/pkg/types"
	"golang.org/x/sync/errgroup"
)

const (
	LabelSysCtr   = "sh.tmacro.sysctr"
	LabelName     = "sh.tmacro.sysctr.name"
	LabelSpecHash = "sh.tmacro.sysctr.specHash"
)

type RunOptions struct {
	Cleanup bool
}

func Run(ctx context.Context, drv driver.Driver, spec *types.Spec, opts RunOptions) (int, error) {
	g, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)

	idChan := make(chan string, 1)
	defer close(idChan)

	var exitCode int

	g.Go(func() error {
		containerID, err := run(ctx, drv, spec)
		if err != nil {
			idChan <- ""
			return err
		}

		idChan <- containerID

		for {
			select {
			case <-ctx.Done():
				stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				err := drv.StopContainer(stopCtx, containerID)
				if err != nil {
					return fmt.Errorf("failed to stop container: %w", err)
				}

				if opts.Cleanup {
					remCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					err = drv.RemoveContainer(remCtx, containerID)
					if err != nil {
						return fmt.Errorf("failed to remove container: %w", err)
					}
				}

				return nil
			default:
				err = drv.WaitForExit(ctx, containerID)
				if errors.Is(err, context.Canceled) {
					continue
				}
				if err != nil {
					return fmt.Errorf("failed to wait for container to exit: %w", err)
				}

				cancel()

				statusCtx, statusCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer statusCancel()

				status, err := drv.ContainerStatus(statusCtx, containerID)
				if err != nil {
					return fmt.Errorf("failed to get container status: %w", err)
				}

				zerolog.Ctx(ctx).Info().Str("id", containerID).Int("exit_code", status.ExitCode).Msg("container exited")
				exitCode = status.ExitCode

				return nil
			}
		}
	})

	g.Go(func() error {
		select {
		case <-ctx.Done():
			return nil
		case containerID := <-idChan:
			if containerID == "" {
				return nil
			}
			for {
				select {
				case <-ctx.Done():
					return nil
				default:
					err := drv.GetLogs(ctx, containerID, os.Stdout, os.Stderr)
					if err != nil && !errors.Is(err, context.Canceled) {
						return fmt.Errorf("failed to get logs: %w", err)
					}
				}
			}
		}
	})

	err := g.Wait()
	return exitCode, err
}

func hashSpec(spec *types.Spec) string {
	h := sha256.New()

	h.Write([]byte(spec.Name + "\n"))
	h.Write([]byte(spec.Image + "\n"))

	for _, m := range spec.Command {
		h.Write([]byte(m + "\n"))
	}

	for _, m := range spec.Args {
		h.Write([]byte(m + "\n"))
	}

	envVarNames := make([]string, len(spec.Env))
	envVars := make(map[string]string, len(spec.Env))

	for i, m := range spec.Env {
		envVars[m.Name] = m.Value
		envVarNames[i] = m.Name
	}

	sort.Strings(envVarNames)

	for _, name := range envVarNames {
		h.Write([]byte(fmt.Sprintf("%s=%s\n", name, envVars[name])))
	}

	return hex.EncodeToString(h.Sum(nil))
}

func run(ctx context.Context, drv driver.Driver, spec *types.Spec) (string, error) {
	logger := zerolog.Ctx(ctx)

	status, err := drv.FindContainer(ctx, spec.Name, map[string]string{
		LabelSysCtr: "true",
		LabelName:   spec.Name,
	})

	if err != nil && !errors.Is(err, driver.ErrContainerNotFound) {
		return "", fmt.Errorf("failed to fetch containers: %w", err)
	}

	configHash := hashSpec(spec)

	containerID := ""
	needsStart := true

	if status != nil {
		containerID = status.ID
		needsRemoval := status.Status != driver.Running

		if status.Status == driver.Running {
			hashLabel, ok := status.Labels[LabelSpecHash]
			fmt.Println(configHash)
			fmt.Println(hashLabel)
			fmt.Println(ok)
			if !ok || hashLabel != configHash {
				needsRemoval = true
				logger.Info().Str("id", containerID).Msg("recreating container")
				err = drv.StopContainer(ctx, status.ID)
				if err != nil {
					return "", fmt.Errorf("failed to stop container: %w", err)
				}
			} else {
				needsStart = false
				logger.Info().Str("id", containerID).Msg("attaching to running container")
			}
		}

		if needsRemoval {
			logger.Info().Str("id", containerID).Msg("removing container")
			err = drv.RemoveContainer(ctx, status.ID)
			if err != nil {
				return "", fmt.Errorf("failed to remove container: %w", err)
			}
			containerID = ""
		}
	}

	if containerID == "" {
		env := make(map[string]string, len(spec.Env))
		for _, e := range spec.Env {
			env[e.Name] = e.Value
		}

		volumes := make([]driver.Volume, len(spec.VolumeMounts))
		for i, v := range spec.VolumeMounts {
			ro := v.ReadOnly != nil && *v.ReadOnly
			volumes[i] = driver.Volume{
				Source:   v.Source,
				Target:   v.Target,
				ReadOnly: ro,
			}
		}

		containerID, err = drv.CreateContainer(ctx, &driver.Spec{
			Name:        spec.Name,
			Image:       spec.Image,
			Command:     spec.Command,
			Arguments:   spec.Args,
			Environment: env,
			Labels: map[string]string{
				LabelSysCtr:   "true",
				LabelName:     spec.Name,
				LabelSpecHash: configHash,
			},
			Volumes: volumes,
		})

		if err != nil {
			return "", fmt.Errorf("failed to create container: %w", err)
		}

		logger.Info().Str("id", containerID).Msg("created container")
	}

	if needsStart {
		err = drv.StartContainer(ctx, containerID)
		if err != nil {
			return "", fmt.Errorf("failed to start container: %w", err)
		}

		logger.Info().Str("id", containerID).Msg("container started")
	}

	return containerID, nil
}
