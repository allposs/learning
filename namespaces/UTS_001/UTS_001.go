package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	var nsShell string
	flag.StringVar(&nsShell, "nsshell", "/bin/sh", "Path to the shell to use")
	flag.Parse()
	nsRun(nsShell)
}
func nsRun(command string) {
	log.Println(command)
	cmd := exec.Command(command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS,
	}
	if err := cmd.Run(); err != nil {
		log.Printf("Error running the /bin/sh command - %s\n", err)
		os.Exit(1)
	}

}
