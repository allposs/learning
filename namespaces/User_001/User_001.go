package main

import (
	"flag"
	"os"
	"os/exec"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
	"github.com/sirupsen/logrus"
)

func init() {
	reexec.Register("nsInitialisation", nsInit)
	if reexec.Init() {
		os.Exit(0)
	}
}

func main() {
	var nsShell, nsHostName, rootPath string
	flag.StringVar(&nsShell, "nsshell", "/bin/sh", "The path to the shell where the namespace is running")
	flag.StringVar(&nsHostName, "nshostname", "nshost", "Path to the shell to use")
	flag.StringVar(&rootPath, "rootfs", "/tmp/busybox", "Path to the root filesystem to use")
	flag.Parse()
	cmd := reexec.Command("nsInitialisation", nsShell, nsHostName, rootPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWUSER,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
	}
	check(cmd.Run())
}

//nsInit ns初始化
func nsInit() {
	command := os.Args[1]
	hostname := os.Args[2]
	newRootPath := os.Args[3]
	mountRoot(newRootPath)
	chRoot(command, newRootPath)
	nsRun(command, hostname, newRootPath)
}

func nsRun(command, hostname, newRootPath string) {
	cmd := exec.Command(command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	check(syscall.Sethostname([]byte(hostname)))
	check(cmd.Run())
	check(syscall.Unmount("proc", 0))
	check(syscall.Unmount("temp", 0))
}

func mountRoot(newroot string) {
	check(syscall.Chroot(newroot))
	check(os.Chdir("/"))
}

func chRoot(command, newroot string) {
	cmd := exec.Command(command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	check(syscall.Mount("proc", "proc", "proc", 0, ""))
	check(syscall.Mount("godir", "temp", "tmpfs", 0, ""))
}

func check(err error) {
	if err != nil {
		logrus.Errorln(err)
	}
}
