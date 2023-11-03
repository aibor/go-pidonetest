package qemu_test

import (
	"testing"

	"github.com/aibor/pidonetest/internal/qemu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArgs(t *testing.T) {
	next := func(s *[]string) string {
		e := (*s)[0]
		*s = (*s)[1:]
		return e
	}

	t.Run("yes-kvm", func(t *testing.T) {
		q := qemu.Command{}

		assert.Contains(t, q.Args(), "-enable-kvm")
	})

	t.Run("no-kvm", func(t *testing.T) {
		q := qemu.Command{
			NoKVM: true,
		}

		assert.NotContains(t, q.Args(), "-enable-kvm")
	})

	t.Run("yes-verbose", func(t *testing.T) {
		q := qemu.Command{
			Verbose: true,
		}

		assert.NotContains(t, q.Args()[len(q.Args())-1], "loglevel=0")
	})

	t.Run("no-verbose", func(t *testing.T) {
		q := qemu.Command{}

		assert.Contains(t, q.Args()[len(q.Args())-1], "loglevel=0")
	})

	t.Run("serial files virtio-mmio", func(t *testing.T) {
		q := qemu.Command{
			ExtraFiles: []string{
				"/output/file1",
				"/output/file2",
			},
			TransportType: qemu.TransportTypeMMIO,
		}
		args := q.Args()
		expected := []string{
			"file,id=vcon1,path=/dev/fd/1",
			"file,id=vcon3,path=/dev/fd/3",
			"file,id=vcon4,path=/dev/fd/4",
		}

		for len(args) > 1 {
			arg := next(&args)
			if arg != "-chardev" {
				continue
			}
			if assert.Greater(t, len(expected), 0, "expected serial files already consumed") {
				assert.Equal(t, next(&expected), next(&args))
			}
		}

		assert.Len(t, expected, 0, "no expected serial files should be left over")
	})

	t.Run("serial files isa-pci", func(t *testing.T) {
		q := qemu.Command{
			ExtraFiles: []string{
				"/output/file1",
				"/output/file2",
			},
			TransportType: qemu.TransportTypeISA,
		}
		args := q.Args()
		expected := []string{
			"file:/dev/fd/1",
			"file:/dev/fd/3",
			"file:/dev/fd/4",
		}

		for len(args) > 1 {
			arg := next(&args)
			if arg != "-serial" {
				continue
			}
			if assert.Greater(t, len(expected), 0, "expected serial files already consumed") {
				assert.Equal(t, next(&expected), next(&args))
			}
		}

		assert.Len(t, expected, 0, "no expected serial files should be left over")
	})

	t.Run("init args", func(t *testing.T) {
		q := qemu.Command{
			InitArgs: []string{
				"first",
				"second",
				"third",
			},
		}
		args := q.Args()
		expected := " -- first second third"

		var appendValue string
		for len(args) > 1 {
			arg := next(&args)
			if arg == "-append" {
				appendValue = next(&args)
			}
		}

		require.NotEmpty(t, appendValue, "append value must be found")
		assert.Contains(t, appendValue, expected, "append value should contain init args")
	})
}
