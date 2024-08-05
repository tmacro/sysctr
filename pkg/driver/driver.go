package driver

import (
	"context"
	"errors"
	"io"
)

type Driver interface {
	DriverInfo() DriverInfo

	PullImage(ctx context.Context, image string) error
	FindContainer(ctx context.Context, name string, labels map[string]string) (*Status, error)
	ContainerStatus(ctx context.Context, id string) (*Status, error)
	CreateContainer(ctx context.Context, spec *Spec) (string, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string) error
	RemoveContainer(ctx context.Context, id string) error
	WaitForExit(ctx context.Context, id string) error
	GetLogs(ctx context.Context, id string, stdout, stderr io.Writer) error
}

type DriverInfo struct {
	ID  string
	New func() Driver
}

type Provisioner interface {
	Provision(ctx context.Context) error
}

type Destructor interface {
	Destroy(ctx context.Context) error
}

type Spec struct {
	Name        string
	Image       string
	Labels      map[string]string
	Command     []string
	Arguments   []string
	Environment map[string]string
	Volumes     []Volume
}

type Volume struct {
	Source   string
	Target   string
	ReadOnly bool
}

type ContainerStatus string

func (s ContainerStatus) String() string {
	return string(s)
}

const (
	UnknownStatus ContainerStatus = ""
	Created       ContainerStatus = "created"
	Running       ContainerStatus = "running"
	Stopped       ContainerStatus = "stopped"
)

type Status struct {
	ID       string
	Status   ContainerStatus
	Labels   map[string]string
	ExitCode int
}

var (
	ErrContainerNotFound = errors.New("container not found")
)
