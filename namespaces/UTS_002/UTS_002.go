package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	var nsShell, nsHostName string
	flag.StringVar(&nsShell, "nsshell", "/bin/sh", "The path to the shell where the namespace is running")
	flag.StringVar(&nsHostName, "nshostname", "nshost", "Path to the shell to use")
	flag.Parse()
	nsRun(nsShell, nsHostName)
}
func nsRun(command, hostname string) {
	cmd := exec.Command(command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS,
	}
	if err := syscall.Sethostname([]byte(hostname)); err != nil {
		fmt.Printf("Error setting hostname - %s\n", err)
		os.Exit(1)
	}
	if err := cmd.Run(); err != nil {
		log.Printf("Error running the /bin/sh command - %s\n", err)
		os.Exit(1)
	}

}
