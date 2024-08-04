// SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/aibor/virtrun/internal"
	"github.com/aibor/virtrun/internal/qemu"
)

func setupLogging(debug bool) {
	level := slog.LevelWarn
	if debug {
		level = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{
			Level: level,
		},
	)))
}

func run() error {
	arch, err := internal.GetArch()
	if err != nil {
		return fmt.Errorf("get arch: %w", err)
	}

	args, err := internal.NewArgs(arch)
	if err != nil {
		return fmt.Errorf("new args: %w", err)
	}

	err = args.ParseArgs(
		os.Args[0],
		internal.PrependEnvArgs(os.Args[1:]),
		os.Stderr,
	)
	if err != nil {
		return fmt.Errorf("parse args: %w", err)
	}

	setupLogging(args.Debug)

	err = args.Validate()
	if err != nil {
		return fmt.Errorf("validate args: %w", err)
	}

	// Build initramfs for the run.
	irfs, err := internal.NewInitramfsArchive(args.InitramfsArgs)
	if err != nil {
		return fmt.Errorf("initramfs: %w", err)
	}

	slog.Debug("Initramfs created", slog.String("path", irfs.Path))

	defer func() {
		err := irfs.Cleanup()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cleanup initramfs archive: %v", err)
		}

		slog.Debug("Initramfs cleaned up", slog.String("path", irfs.Path))
	}()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGABRT,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)
	defer cancel()

	cmd, err := internal.NewQemuCommand(ctx, args.QemuArgs, irfs.Path)
	if err != nil {
		return fmt.Errorf("build qemu command: %w", err)
	}

	slog.Debug("QEMU command",
		slog.String("command", cmd.String()),
	)

	err = cmd.Run(os.Stdout, os.Stderr)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}

	return nil
}

func handleRunError(err error, errWriter io.Writer) int {
	if err == nil {
		return 0
	}

	// [flag.ErrHelp] is returned when help is requested. So exit without error
	// in this case.
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}

	exitCode := -1

	// ParseArgs already prints errors, so we just exit without an error.
	if errors.Is(err, &internal.ParseArgsError{}) {
		return exitCode
	}

	var qemuCmdErr *qemu.CommandError

	if errors.As(err, &qemuCmdErr) {
		if qemuCmdErr.ExitCode != 0 {
			exitCode = qemuCmdErr.ExitCode
		}
	}

	// Do not print the error in case the guest process ran successfully and
	// the guest properly communicated a non-zero exit code.
	if errors.Is(err, qemu.ErrGuestNonZeroExitCode) {
		return exitCode
	}

	fmt.Fprintf(errWriter, "Error: %v\n", err)

	return exitCode
}

func main() {
	err := run()
	exitCode := handleRunError(err, os.Stderr)
	os.Exit(exitCode)
}
