package driver

import (
	"context"
	"fmt"
	"io"
	"strings"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/errdefs"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/tmacro/sysctr/pkg/driver"
)

func init() {
	driver.RegisterDriver(&ContainerdDriver{})
}

const (
	defaultEndpoint    = "/run/containerd/containerd.sock"
	defaultNamespace   = "sysctr"
	containerNameLabel = "sysctr.driver.containerd.name"
)

type ContainerdDriver struct {
	Namespace string `json:"namespace"`
	Endpoint  string `json:"endpoint"`

	client *containerd.Client
}

func (d *ContainerdDriver) DriverInfo() driver.DriverInfo {
	return driver.DriverInfo{
		ID:  "containerd",
		New: func() driver.Driver { return new(ContainerdDriver) },
	}
}

func (d *ContainerdDriver) Provision(ctx context.Context) error {
	var err error

	if d.Endpoint == "" {
		d.Endpoint = defaultEndpoint
	}

	if d.Namespace == "" {
		d.Namespace = defaultNamespace
	}

	d.client, err = containerd.New(d.Endpoint)
	if err != nil {
		return err
	}

	return nil
}

func (d *ContainerdDriver) Destroy(ctx context.Context) error {
	if d.client != nil {
		return d.client.Close()
	}

	return nil
}

func resolveImageRef(imageRef string) string {
	if !strings.Contains(imageRef, ":") {
		imageRef = imageRef + ":latest"
	}

	slashes := strings.Count(imageRef, "/")
	if slashes == 0 {
		imageRef = "docker.io/library/" + imageRef
	} else if slashes == 1 {
		imageRef = "docker.io/" + imageRef
	}

	return imageRef
}

func (d *ContainerdDriver) PullImage(ctx context.Context, imageRef string) error {
	ctx = namespaces.WithNamespace(ctx, d.Namespace)

	imageRef = resolveImageRef(imageRef)

	_, err := d.client.Pull(ctx, imageRef, containerd.WithPullUnpack)
	if err != nil {
		return err
	}
	return nil
}

func (d *ContainerdDriver) CreateContainer(ctx context.Context, spec *driver.Spec) (string, error) {
	ctx = namespaces.WithNamespace(ctx, d.Namespace)
	img, err := d.client.GetImage(ctx, resolveImageRef(spec.Image))
	if err != nil {
		return "", err
	}

	args := make([]string, 0)
	args = append(args, spec.Command...)
	args = append(args, spec.Arguments...)

	env := make([]string, 0)
	for k, v := range spec.Environment {
		env = append(env, k+"="+v)
	}

	mounts := make([]specs.Mount, 0)
	for _, v := range spec.Volumes {
		mode := "rw"
		if v.ReadOnly {
			mode = "ro"
		}

		mounts = append(mounts, specs.Mount{
			Type:        "bind",
			Source:      v.Source,
			Destination: v.Target,
			Options:     []string{"rbind", mode},
		})
	}

	labels := make(map[string]string, len(spec.Labels)+1)
	for k, v := range spec.Labels {
		labels[k] = v
	}

	container, err := d.client.NewContainer(
		ctx,
		spec.Name,
		containerd.WithNewSnapshot(spec.Name+"-snapshot", img),
		containerd.WithNewSpec(
			oci.WithImageConfig(img),
			oci.WithProcessArgs(args...),
			oci.WithEnv(env),
			oci.WithMounts(mounts),
		),
		containerd.WithAdditionalContainerLabels(spec.Labels),
	)

	if err != nil {
		return "", err
	}

	return container.ID(), nil
}

func (d *ContainerdDriver) StartContainer(ctx context.Context, id string) error {
	ctx = namespaces.WithNamespace(ctx, d.Namespace)
	container, err := d.client.LoadContainer(ctx, id)
	if err != nil {
		return err
	}

	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return err
	}

	err = task.Start(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (d *ContainerdDriver) StopContainer(ctx context.Context, id string) error {
	ctx = namespaces.WithNamespace(ctx, d.Namespace)
	container, err := d.client.LoadContainer(ctx, id)
	if err != nil {
		return err
	}

	task, err := container.Task(ctx, nil)
	if err != nil && !errdefs.IsNotFound(err) {
		return err
	} else if errdefs.IsNotFound(err) {
		return nil
	}

	return task.Kill(ctx, syscall.SIGTERM, containerd.WithKillAll)
}

func getNameFromLabels(ctx context.Context, container containerd.Container) string {
	labels, err := container.Labels(ctx)
	if err != nil {
		return ""
	}

	return labels[containerNameLabel]
}

func (d *ContainerdDriver) RemoveContainer(ctx context.Context, id string) error {
	ctx = namespaces.WithNamespace(ctx, d.Namespace)
	container, err := d.client.LoadContainer(ctx, id)
	if err != nil {
		return err
	}

	task, err := container.Task(ctx, nil)
	if err != nil && !errdefs.IsNotFound(err) {
		return err
	}

	if err == nil {
		_, err = task.Delete(ctx)
		if err != nil {
			return err
		}
	}

	name := getNameFromLabels(ctx, container)
	err = d.client.SnapshotService(containerd.DefaultSnapshotter).Remove(ctx, name+"-snapshot")
	if err != nil {
		return err
	}

	err = container.Delete(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (d *ContainerdDriver) WaitForExit(ctx context.Context, id string) error {
	ctx = namespaces.WithNamespace(ctx, d.Namespace)
	container, err := d.client.LoadContainer(ctx, id)
	if err != nil {
		return err
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		return err
	}

	statusC, err := task.Wait(ctx)
	if err != nil {
		return err
	}

	<-statusC
	return nil
}

func (d *ContainerdDriver) GetLogs(ctx context.Context, id string, stdout, stderr io.Writer) error {
	return d.WaitForExit(ctx, id)
}

func (d *ContainerdDriver) FindContainer(ctx context.Context, name string, labels map[string]string) (*driver.Status, error) {
	ctx = namespaces.WithNamespace(ctx, d.Namespace)

	filters := []string{"name==" + name}
	for k, v := range labels {
		filters = append(filters, "labels."+k+"=="+v)
	}

	containers, err := d.client.Containers(ctx, filters...)
	if err != nil {
		return nil, err
	}

	if len(containers) == 0 {
		return nil, driver.ErrContainerNotFound
	}

	if len(containers) > 1 {
		return nil, fmt.Errorf("found multiple containers with name %s", name)
	}

	container := containers[0]

	containerLabels, err := container.Labels(ctx)
	if err != nil {
		return nil, err
	}

	ec, taskStatus, err := getTaskStatus(ctx, container)
	if err != nil {
		return nil, err
	}

	status := driver.Status{
		ID:       container.ID(),
		Status:   taskStatus,
		Labels:   containerLabels,
		ExitCode: ec,
	}

	return &status, nil
}

func (d *ContainerdDriver) ContainerStatus(ctx context.Context, id string) (*driver.Status, error) {
	ctx = namespaces.WithNamespace(ctx, d.Namespace)

	container, err := d.client.LoadContainer(ctx, id)
	if err != nil {
		return nil, err
	}

	containerLabels, err := container.Labels(ctx)
	if err != nil {
		return nil, err
	}

	ec, taskStatus, err := getTaskStatus(ctx, container)
	if err != nil {
		return nil, err
	}

	status := driver.Status{
		ID:       container.ID(),
		Status:   taskStatus,
		Labels:   containerLabels,
		ExitCode: ec,
	}

	return &status, nil
}

func getTaskStatus(ctx context.Context, container containerd.Container) (int, driver.ContainerStatus, error) {
	task, err := container.Task(ctx, nil)
	if err != nil && !errdefs.IsNotFound(err) {
		return 0, driver.UnknownStatus, err
	}

	if errdefs.IsNotFound(err) {
		return 0, driver.Stopped, nil
	}

	status, err := task.Status(ctx)
	if err != nil {
		return 0, driver.UnknownStatus, err
	}

	return int(status.ExitStatus), translateContainerdStatus(status.Status), nil
}

func translateContainerdStatus(status containerd.ProcessStatus) driver.ContainerStatus {
	switch status {
	case containerd.Created:
		return driver.Created
	case containerd.Running:
		return driver.Running
	case containerd.Pausing:
		return driver.Running
	case containerd.Paused:
		return driver.Stopped
	case containerd.Stopped:
		return driver.Stopped
	default:
		return driver.UnknownStatus
	}
}
