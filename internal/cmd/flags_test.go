// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd_test

import (
	"io"
	"testing"

	"github.com/aibor/virtrun/internal/cmd"
	"github.com/aibor/virtrun/internal/qemu"
	"github.com/aibor/virtrun/internal/virtrun"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlags_ParseArgs(t *testing.T) {
	absBinPath, err := cmd.AbsoluteFilePath("bin.test")
	require.NoError(t, err)

	tests := []struct {
		name              string
		args              []string
		expectedSpec      *virtrun.Spec
		expectedDebugFlag bool
		expecterErr       error
	}{
		{
			name: "help",
			args: []string{
				"-help",
			},
			expecterErr: cmd.ErrHelp,
		},
		{
			name: "version",
			args: []string{
				"-version",
			},
			expecterErr: cmd.ErrHelp,
		},
		{
			name: "no kernel",
			args: []string{
				"bin.test",
			},
			expecterErr: &cmd.ParseArgsError{},
		},
		{
			name: "no binary",
			args: []string{
				"-kernel=/boot/this",
			},
			expecterErr: &cmd.ParseArgsError{},
		},
		{
			name: "additional file is empty",
			args: []string{
				"-kernel=/boot/this",
				"-addFile=",
				"bin.test",
			},
			expecterErr: &cmd.ParseArgsError{},
		},
		{
			name: "debug",
			args: []string{
				"-kernel=/boot/this",
				"-debug",
				"bin.test",
			},
			expectedSpec: &virtrun.Spec{
				Initramfs: virtrun.Initramfs{
					Binary: absBinPath,
				},
				Qemu: virtrun.Qemu{
					Kernel:   "/boot/this",
					InitArgs: []string{},
				},
			},
			expectedDebugFlag: true,
		},
		{
			name: "simple go test invocation",
			args: []string{
				"-kernel=/boot/this",
				"bin.test",
				"-test.paniconexit0",
				"-test.v=true",
				"-test.timeout=10m0s",
			},
			expectedSpec: &virtrun.Spec{
				Initramfs: virtrun.Initramfs{
					Binary: absBinPath,
				},
				Qemu: virtrun.Qemu{
					Kernel: "/boot/this",
					InitArgs: []string{
						"-test.paniconexit0",
						"-test.v=true",
						"-test.timeout=10m0s",
					},
				},
			},
		},
		{
			name: "go test invocation with virtrun flags",
			args: []string{
				"-kernel=/boot/this",
				"-cpu", "host",
				"-machine=pc",
				"-transport", "mmio",
				"-memory=269",
				"-verbose",
				"-smp", "7",
				"-nokvm=true",
				"-standalone",
				"-noGoTestFlagRewrite",
				"-keepInitramfs",
				"-addFile", "/file2",
				"-addFile", "/dir/file3",
				"bin.test",
				"-test.paniconexit0",
				"-test.v=true",
				"-test.timeout=10m0s",
			},
			expectedSpec: &virtrun.Spec{
				Initramfs: virtrun.Initramfs{
					Binary: absBinPath,
					Files: []string{
						"/file2",
						"/dir/file3",
					},
					StandaloneInit: true,
					Keep:           true,
				},
				Qemu: virtrun.Qemu{
					Kernel:        "/boot/this",
					CPU:           "host",
					Machine:       "pc",
					TransportType: qemu.TransportTypeMMIO,
					Memory:        269,
					NoKVM:         true,
					SMP:           7,
					InitArgs: []string{
						"-test.paniconexit0",
						"-test.v=true",
						"-test.timeout=10m0s",
					},
					Verbose:             true,
					NoGoTestFlagRewrite: true,
				},
			},
		},
		{
			name: "flag parsing stops at flags after binary file",
			args: []string{
				"-kernel=/boot/this",
				"bin.test",
				"-test.paniconexit0",
				"another.file",
				"-x",
				"-standalone",
			},
			expectedSpec: &virtrun.Spec{
				Initramfs: virtrun.Initramfs{
					Binary: absBinPath,
				},
				Qemu: virtrun.Qemu{
					Kernel: "/boot/this",
					InitArgs: []string{
						"-test.paniconexit0",
						"another.file",
						"-x",
						"-standalone",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &virtrun.Spec{}
			flags := cmd.NewFlags("test", spec, io.Discard)

			err := flags.ParseArgs(tt.args)
			require.ErrorIs(t, err, tt.expecterErr)

			if tt.expecterErr != nil {
				return
			}

			assert.Equal(t, tt.expectedSpec, spec, "spec")
			assert.Equal(t, tt.expectedDebugFlag, flags.Debug(), "debug flag")
		})
	}
}
