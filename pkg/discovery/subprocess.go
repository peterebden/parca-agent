// Copyright 2021 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package discovery

import (
	"context"
	"os"
	"os/exec"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/common/model"

	"github.com/parca-dev/parca-agent/pkg/target"
)

// A SubprocessConfig configures a new SubprocessDiscoverer instance.
type SubprocessConfig struct {
	command []string
	cgexec  string
}

// A SubprocessDiscoverer 'discovers' a process to instrument by starting it as a subprocess.
type SubprocessDiscoverer struct {
	logger  log.Logger
	command []string
	cgexec  string
}

func (c *SubprocessConfig) Name() string {
	return c.command
}

// NewSubprocessConfig returns a new config based on the given command + arguments.
func NewSubprocessConfig(command ...string) *SubprocessConfig {
	return &SubprocessConfig{
		command: command,
		cgexec:  "cgexec",
	}
}

// NewDiscoverer creates a new Discoverer from this config.
func (c *SubprocessConfig) NewDiscoverer(d DiscovererOptions) (Discoverer, error) {
	return &SubprocessDiscoverer{
		logger:  d.Logger,
		command: c.command,
		cgexec:  c.cgexec,
	}, nil
}

// Run starts the subprocess and runs this discoverer against it.
func (d *SubprocessDiscoverer) Run(ctx context.Context, up chan<- []*target.Group) error {
	args := append([]string{"-g", "*:" + group}, d.command...)
	level.Debug(d.logger).Log("msg", "starting subprocess", "command", d.command, "args", d.args)
	cmd := exec.CommandContext(ctx, d.cgexec, d.command...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	up <- []*target.Group{{
		Targets: []model.LabelSet{},
		Source:  "",
	}}
	ch := make(chan error)
	go func() {
		ch <- cmd.Wait()
	}()
	select {
	case <-ctx.Done():
		// Don't worry about the command's error here; it will just be that it was killed by
		// CommandContext once the context was done.
		return ctx.Err()
	case err := <-ch:
		return err
	}
}
