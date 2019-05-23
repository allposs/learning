package main

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/sirupsen/logrus"
)

func check(err error) {
	if err != nil {
		logrus.Errorln(err)
	}
}

func run() {
	logrus.Info("Setting up...")
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
	}
	check(cmd.Run())
}

func child() {
	logrus.Infof("Running %v", os.Args[2:])
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	check(syscall.Sethostname([]byte("newhost")))
	// "/root/go/src/PID_001/busybox" busybox解压的目录
	check(syscall.Chroot("/root/go/src/PID_001/busybox"))
	check(os.Chdir("/"))
	// func Mount(source string, target string, fstype string, flags uintptr, data string) (err error)
	// 前三个参数分别是文件系统的名字，挂载到的路径，文件系统的类型
	check(syscall.Mount("proc", "proc", "proc", 0, ""))
	// 这里godir是挂载文件系统的名称，可以修改特殊一些，以方便区分
	check(syscall.Mount("godir", "temp", "tmpfs", 0, ""))
	check(cmd.Run())
	// 卸载
	check(syscall.Unmount("proc", 0))
	check(syscall.Unmount("godir", 0))
}

func main() {
	if len(os.Args) < 2 {
		logrus.Errorf("missing commands")
		return
	}
	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	default:
		logrus.Errorf("wrong command")
		return
	}
}
