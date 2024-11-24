// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package virtrun

import (
	"fmt"
	"os/exec"
)

// Validate file parameters of the given [Spec].
func Validate(spec *Spec) error {
	// Check files are actually present.
	_, err := exec.LookPath(spec.Qemu.Executable)
	if err != nil {
		return fmt.Errorf("qemu binary: %w", err)
	}

	err = spec.Qemu.Kernel.Validate()
	if err != nil {
		return fmt.Errorf("kernel file: %w", err)
	}

	for _, file := range spec.Initramfs.Files {
		err := (*FilePath)(&file).Validate()
		if err != nil {
			return fmt.Errorf("additional file: %w", err)
		}
	}

	for _, file := range spec.Initramfs.Modules {
		err := (*FilePath)(&file).Validate()
		if err != nil {
			return fmt.Errorf("module: %w", err)
		}
	}

	err = spec.Initramfs.Binary.Validate()
	if err != nil {
		return fmt.Errorf("main binary: %w", err)
	}

	return nil
}
