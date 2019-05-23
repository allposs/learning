package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/reexec"
	"github.com/sirupsen/logrus"
)

func init() {
	reexec.Register("nsInitialisation", nsInit)
	if reexec.Init() {
		os.Exit(0)
	}
}

func check(err error) {
	if err != nil {
		logrus.Errorln(err)
	}
}

func main() {
	var nsShell, nsHostName, rootPath, netsetgoPath string
	flag.StringVar(&nsShell, "nsshell", "/bin/sh", "The path to the shell where the namespace is running")
	flag.StringVar(&nsHostName, "nshostname", "nshost", "Path to the shell to use")
	flag.StringVar(&rootPath, "rootfs", "/tmp/busybox", "Path to the root filesystem to use")
	flag.StringVar(&netsetgoPath, "netsetgo", "/tmp/netsetgo", "Path to the netsetgo binary")
	flag.Parse()
	cmd := reexec.Command("nsInitialisation", nsShell, nsHostName, rootPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNET |
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

	check(cmd.Start())
	//netsetgo 必须使用root权限运行，而且要注意setuid的权限
	pid := fmt.Sprintf("%d", cmd.Process.Pid)
	netsetgoCmd := exec.Command(netsetgoPath, "-pid", pid)
	check(netsetgoCmd.Run())
	check(cmd.Wait())

}

//nsInit ns初始化
func nsInit() {
	command := os.Args[1]
	hostname := os.Args[2]
	newRootPath := os.Args[3]
	mountRoot(newRootPath)
	chRoot(command, newRootPath)
	check(waitForNetwork())
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
}

func waitForNetwork() error {
	maxWait := time.Second * 3
	checkInterval := time.Second
	timeStarted := time.Now()
	for {
		interfaces, err := net.Interfaces()
		if err != nil {
			return err
		}
		// pretty basic check ...
		// > 1 as a lo device will already exist
		if len(interfaces) > 1 {
			return nil
		}
		if time.Since(timeStarted) > maxWait {
			return fmt.Errorf("Timeout after %s waiting for network", maxWait)
		}
		time.Sleep(checkInterval)
	}
}
