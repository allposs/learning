package main

import (
	"flag"
	"os"
	"os/exec"
	"syscall"

	"github.com/sirupsen/logrus"
)

func main() {
	var nsShell, nsHostName, rootPath string
	flag.StringVar(&nsShell, "nsshell", "/bin/sh", "The path to the shell where the namespace is running")
	flag.StringVar(&nsHostName, "nshostname", "nshost", "Path to the shell to use")
	flag.StringVar(&rootPath, "rootfs", "/tmp/busybox", "Path to the root filesystem to use")
	flag.Parse()
	switch os.Args[1] {
	case "run":
		nsRun(nsShell, nsHostName, rootPath)
	case "child":
		chRoot(nsShell, rootPath)
	default:
		logrus.Errorf("wrong command")
		return
	}

}

//nsInit ns初始化
func nsInit(command, hostname, newRootPath string) {
	//check(mountRoot(newRootPath))
	nsRun(command, hostname, newRootPath)
}

func nsRun(command, hostname, newRootPath string) {
	cmd := exec.Command("/proc/self/exe", "child")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
	}
	check(syscall.Sethostname([]byte(hostname)))
	check(cmd.Run())
}

func chRoot(command, newroot string) {
	cmd := exec.Command(command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	check(syscall.Chroot(newroot))
	check(os.Chdir("/"))
	check(syscall.Mount("proc", "proc", "proc", 0, ""))
	check(syscall.Mount("godir", "temp", "tmpfs", 0, ""))
	check(cmd.Run())
	check(syscall.Unmount("proc", 0))
	check(syscall.Unmount("temp", 0))
}

func check(err error) {
	if err != nil {
		logrus.Errorln(err)
	}
}
