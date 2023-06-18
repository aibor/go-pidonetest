package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

var debugLog *log.Logger

func init() {
	debugLog = log.New(io.Discard, "DEBUG: ", log.LstdFlags)
}

func run(qemuCmd *QEMUCommand, testBinaryPath string) (int, error) {
	libs, err := resolveLinkedLibs(testBinaryPath)
	if err != nil {
		return 1, err
	}
	if len(libs) > 0 {
		return 1, fmt.Errorf("Test binary must not be linked, but is linked to: % s. Try with CGO_ENABLED=0", libs)
	}

	additional := strings.Split(LibSearchPaths, ":")
	additional = append(additional, libs...)
	initrdFilePath, err := createInitrd(testBinaryPath, additional...)
	if err != nil {
		return 1, fmt.Errorf("create initrd: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(initrdFilePath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to remove initrd file: %s: %v\n", initrdFilePath, err)
		}
	}()

	qemuCmd.Initrd = initrdFilePath

	cmd := qemuCmd.Cmd()

	debugLog.Printf("qemu cmd: %s", cmd.String())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 1, fmt.Errorf("get stdout: %v", err)
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 1, fmt.Errorf("get stderr: %v", err)
	}
	defer stderr.Close()

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("run qemu: %v", err)
	}
	p := cmd.Process
	if p != nil {
		defer func() {
			_ = p.Kill()
		}()
	}

	done := make(chan bool)
	go func() {
		_ = cmd.Wait()
		close(done)
	}()

	rcStream := make(chan int, 1)
	readGroup := sync.WaitGroup{}
	readGroup.Add(1)
	go func() {
		defer readGroup.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			var rc int
			if _, err := fmt.Sscanf(line, "GO_PIDONETEST_RC: %d", &rc); err != nil {
				fmt.Println(line)
				continue
			}
			if len(rcStream) == 0 {
				rcStream <- rc
			}
		}
	}()

	readGroup.Add(1)
	go func() {
		defer readGroup.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
		}
	}()

	signalStream := make(chan os.Signal, 1)
	signal.Notify(signalStream, syscall.SIGABRT, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)

	rc := 1

	select {
	case sig := <-signalStream:
		return rc, fmt.Errorf("signal received: %d, %s", sig, sig)
	case <-done:
		break
	}

	_ = os.Remove(initrdFilePath)
	readGroup.Wait()
	if len(rcStream) == 1 {
		rc = <-rcStream
	}
	return rc, nil
}

func main() {
	var testBinaryPath string
	var qemuCmd = QEMUCommand{
		Binary:  "qemu-system-x86_64",
		Kernel:  "/boot/vmlinuz-linux",
		Machine: "q35",
		CPU:     "host",
		Memory:  128,
		NoKVM:   false,
	}

	if !parseFlags(os.Args, &qemuCmd, &testBinaryPath) {
		// Flag already prints errors, so we just exit.
		os.Exit(1)
	}

	rc, err := run(&qemuCmd, testBinaryPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}

	os.Exit(rc)
}
