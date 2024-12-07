// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package qemu

import (
	"fmt"
	"slices"
	"strings"
)

const (
	// TransportTypeISA is ISA legacy transport. It should work for amd64 in
	// any case. With "microvm" machine type only provides one console for
	// stdout.
	TransportTypeISA TransportType = "isa"
	// TransportTypePCI is VirtIO PCI transport. Requires kernel built with
	// CONFIG_VIRTIO_PCI.
	TransportTypePCI TransportType = "pci"
	// TransportTypeMMIO is Virtio MMIO transport. Requires kernel built with
	// CONFIG_VIRTIO_MMIO.
	TransportTypeMMIO TransportType = "mmio"
)

// TransportType represents QEMU IO transport types.
type TransportType string

func (t *TransportType) isKnown() bool {
	knownTransportTypes := []TransportType{
		TransportTypeISA,
		TransportTypePCI,
		TransportTypeMMIO,
	}

	return slices.Contains(knownTransportTypes, *t)
}

// String returns the [TransportType]'s underlying string value.
//
// It returns the empty string for unknown [TransportType]s.
func (t *TransportType) String() string {
	if !t.isKnown() {
		return ""
	}

	return string(*t)
}

// Set parses the given string and sets the receiving [TransportType].
//
// It returns ErrTransportTypeInvalid if the string does not represent a valid
// [TransportType].
func (t *TransportType) Set(s string) error {
	tt := TransportType(s)

	if !tt.isKnown() {
		return ErrTransportTypeInvalid
	}

	*t = tt

	return nil
}

// ConsoleDeviceName returns the name of the console device in the guest.
func (t *TransportType) ConsoleDeviceName(num uint) string {
	f := "hvc%d"
	if *t == TransportTypeISA {
		f = "ttyS%d"
	}

	return fmt.Sprintf(f, num)
}

type consoleFunc func(backend string, args ...string) []Argument

func consoleArgsFunc(transportType TransportType) consoleFunc {
	consoleID := 0

	sharedDevices := map[TransportType]string{
		TransportTypePCI:  "virtio-serial-pci,max_ports=8",
		TransportTypeMMIO: "virtio-serial-device,max_ports=8",
	}
	sharedDeviceValue := sharedDevices[transportType]

	return func(backend string, args ...string) []Argument {
		var a []Argument

		if consoleID == 0 && sharedDeviceValue != "" {
			a = append(a, RepeatableArg("device", sharedDeviceValue))
		}

		conID := fmt.Sprintf("con%d", consoleID)
		chardevArgs := []string{backend, "id=" + conID}
		chardevArgs = append(chardevArgs, args...)
		a = append(a, RepeatableArg("chardev", strings.Join(chardevArgs, ",")))

		switch transportType {
		case TransportTypeISA:
			a = append(a, RepeatableArg("serial", "chardev:"+conID))
		case TransportTypePCI, TransportTypeMMIO:
			a = append(a, RepeatableArg("device", "virtconsole,chardev="+conID))
		default: // Ignore invalid transport types.
			return nil
		}

		consoleID++

		return a
	}
}

func fdPath(fd int) string {
	return fmt.Sprintf("/dev/fd/%d", fd)
}
