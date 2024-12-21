// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	"github.com/aibor/virtrun/internal/sys"
)

// Spec describes a single [Run].
//
// It is split into parameters required for the [qemu.CommandSpec] and
// parameters required for building the initramfs archive file.
type Spec struct {
	Qemu      Qemu
	Initramfs Initramfs
}

// Run runs with the given [Spec].
//
// An initramfs archive file is built and used for running QEMU. It returns no
// error if the run succeeds. To succeed, the guest system must explicitly
// communicate exit code 0. The built initramfs archive file is removed, unless
// [Spec.Initramfs.Keep] is set to true.
func Run(
	ctx context.Context,
	spec *Spec,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	arch, err := sys.ReadELFArch(spec.Initramfs.Binary)
	if err != nil {
		return fmt.Errorf("read main binary arch: %w", err)
	}

	err = spec.Qemu.addDefaultsFor(arch)
	if err != nil {
		return err
	}

	initFn := func() (fs.File, error) { return initProgFor(arch) }

	path, removeFn, err := BuildInitramfsArchive(ctx, spec.Initramfs, initFn)
	if err != nil {
		return err
	}
	defer removeFn() //nolint:errcheck

	cmd, err := NewQemuCommand(spec.Qemu, path)
	if err != nil {
		return err
	}

	err = cmd.Run(ctx, stdin, stdout, stderr)
	if err != nil {
		return fmt.Errorf("qemu run: %w", err)
	}

	return nil
}
