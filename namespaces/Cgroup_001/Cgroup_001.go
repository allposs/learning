package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

/*

mkdir /tmp/busybox
tar xf busybox.tar -C /tmp/busybox/
cp netsetgo /tmp/
sudo chown root:root /tmp/netsetgo
sudo chmod 4755 /tmp/netsetgo

*/
func main() {
	var nsShell, nsHostName, rootPath, netsetgoPath string
	flag.StringVar(&nsShell, "nsshell", "/bin/sh", "The path to the shell where the namespace is running")
	flag.StringVar(&nsHostName, "nshostname", "nshost", "Path to the shell to use")
	flag.StringVar(&rootPath, "rootfs", "/tmp/busybox", "Path to the root filesystem to use")
	flag.StringVar(&netsetgoPath, "netsetgo", "/tmp/netsetgo", "Path to the netsetgo binary")
	flag.Parse()
	cg()
	cmd := reexec.Command("nsInitialisation", nsShell, nsHostName, rootPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWIPC |
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
	check(syscall.Mount("godir", "tmp", "tmpfs", 0, ""))

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

/*
#!/bin/sh
d() { /bin/sleep 1000; }
for i in $(seq 1 100)
do
    echo "sleep $i\n"
    d&
done
*/

func cg() {
	cgPath := "/sys/fs/cgroup/"
	pidsPath := filepath.Join(cgPath, "pids")
	// 在/sys/fs/cgroup/pids下创建container目录
	os.Mkdir(filepath.Join(pidsPath, "container"), 0775)
	if !Exists(filepath.Join(pidsPath, "container")) {
		os.MkdirAll(filepath.Join(pidsPath, "container"), os.ModePerm)
		fmt.Printf("file is on:%s", filepath.Join(pidsPath, "container"))
	}
	// 设置最大进程数目为20
	check(ioutil.WriteFile(filepath.Join(pidsPath, "container/pids.max"), []byte("20"), 0777))
	// 将notify_on_release值设为1，当cgroup不再包含任何任务的时候将执行release_agent的内容
	check(ioutil.WriteFile(filepath.Join(pidsPath, "container/notify_on_release"), []byte("1"), 0777))
	// 加入当前正在执行的进程
	fmt.Println(os.Getpid())
	check(ioutil.WriteFile(filepath.Join(pidsPath, "container/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0777))
}

func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
