package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/rs/zerolog"

	dockerContainer "github.com/docker/docker/api/types/container"
	dockerFilters "github.com/docker/docker/api/types/filters"
	dockerImage "github.com/docker/docker/api/types/image"
	dockerMounts "github.com/docker/docker/api/types/mount"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/tmacro/sysctr/pkg/driver"
)

func init() {
	driver.RegisterDriver(&DockerDriver{})
}

type DockerDriver struct {
	client *dockerClient.Client
}

func (d *DockerDriver) DriverInfo() driver.DriverInfo {
	return driver.DriverInfo{
		ID:  "docker",
		New: func() driver.Driver { return new(DockerDriver) },
	}
}

type pullLogLine struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (d *DockerDriver) Provision(ctx context.Context) error {
	var err error
	d.client, err = dockerClient.NewClientWithOpts(dockerClient.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	return nil
}

func (d *DockerDriver) PullImage(ctx context.Context, imageRef string) error {
	resp, err := d.client.ImagePull(ctx, imageRef, dockerImage.PullOptions{})
	if err != nil {
		return err
	}

	defer resp.Close()

	logger := zerolog.Ctx(ctx).With().Str("driver", "docker").Logger()

	digest := ""

	scanner := bufio.NewScanner(resp)
	for scanner.Scan() {
		var line pullLogLine
		err := json.Unmarshal(scanner.Bytes(), &line)
		if err != nil {
			return err
		}

		if line.ID != "" {
			if strings.HasSuffix(imageRef, line.ID) {
				logger.Debug().Str("tag", line.ID).Msg(line.Status)
			}
		} else if line.Status != "" {
			if strings.HasPrefix(line.Status, "Digest:") {
				digest = strings.TrimPrefix(line.Status, "Digest: ")
			}

			if strings.HasPrefix(line.Status, "Status:") {
				status := strings.ToLower(strings.TrimPrefix(line.Status, "Status: "))
				logger.Info().Str("digest", digest).Msg(status)
			}
		}
	}

	return scanner.Err()
}

func (d *DockerDriver) FindContainer(ctx context.Context, name string, labels map[string]string) (*driver.Status, error) {
	filter := dockerFilters.NewArgs()
	filter.Add("name", name)
	for k, v := range labels {
		filter.Add("label", fmt.Sprintf("%s=%s", k, v))
	}

	containers, err := d.client.ContainerList(ctx, dockerContainer.ListOptions{
		All:     true,
		Filters: filter,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		return nil, driver.ErrContainerNotFound
	}

	if len(containers) > 1 {
		return nil, fmt.Errorf("found multiple containers with name %s", name)
	}

	container := containers[0]

	return &driver.Status{
		ID:     container.ID,
		Status: convertDockerStatus(container.State),
		Labels: container.Labels,
	}, nil
}

func convertDockerStatus(status string) driver.ContainerStatus {
	if status == "created" {
		return driver.Created
	}

	if status == "running" || status == "paused" || status == "restarting" {
		return driver.Running
	}

	if status == "exited" || status == "dead" || status == "removing" {
		return driver.Stopped
	}

	return driver.UnknownStatus
}

func (d *DockerDriver) ContainerStatus(ctx context.Context, id string) (*driver.Status, error) {
	container, err := d.client.ContainerInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	containerStatus := convertDockerStatus(container.State.Status)

	if containerStatus == driver.UnknownStatus {
		return nil, fmt.Errorf("unknown container status: %s", container.State.Status)
	}

	status := driver.Status{
		ID:     id,
		Status: containerStatus,
		Labels: container.Config.Labels,
	}

	if container.State.ExitCode != 0 {
		status.ExitCode = container.State.ExitCode
	}

	return &status, nil
}

func convertEnv(env map[string]string) []string {
	if env == nil {
		return []string{}
	}

	var envs []string
	for k, v := range env {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	return envs
}

func (d *DockerDriver) CreateContainer(ctx context.Context, spec *driver.Spec) (string, error) {
	containerConfig := dockerContainer.Config{
		Image:      spec.Image,
		Entrypoint: spec.Command,
		Cmd:        spec.Arguments,
		Env:        convertEnv(spec.Environment),
		Labels:     spec.Labels,
	}

	mounts := make([]dockerMounts.Mount, 0)
	for _, v := range spec.Volumes {
		mounts = append(mounts, dockerMounts.Mount{
			Type:     dockerMounts.TypeBind,
			Source:   v.Source,
			Target:   v.Target,
			ReadOnly: v.ReadOnly,
		})
	}

	hostConfig := dockerContainer.HostConfig{
		NetworkMode: "host",
		Mounts:      mounts,
	}

	container, err := d.client.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, spec.Name)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return container.ID, nil
}

func (d *DockerDriver) StartContainer(ctx context.Context, id string) error {
	return d.client.ContainerStart(ctx, id, dockerContainer.StartOptions{})
}

func (d *DockerDriver) StopContainer(ctx context.Context, id string) error {
	return d.client.ContainerStop(ctx, id, dockerContainer.StopOptions{})
}

func (d *DockerDriver) RemoveContainer(ctx context.Context, id string) error {
	return d.client.ContainerRemove(ctx, id, dockerContainer.RemoveOptions{
		Force: true,
	})
}

func (d *DockerDriver) WaitForExit(ctx context.Context, id string) error {
	evCh, errCh := d.client.ContainerWait(ctx, id, dockerContainer.WaitConditionNotRunning)
	select {
	case <-evCh:
		return nil
	case err := <-errCh:
		return err
	}
}

func (d *DockerDriver) GetLogs(ctx context.Context, id string, stdout, stderr io.Writer) error {
	reader, err := d.client.ContainerLogs(ctx, id, dockerContainer.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return err
	}

	defer reader.Close()

	_, err = stdcopy.StdCopy(stdout, stderr, reader)
	if err != nil {
		return err
	}

	return nil
}
