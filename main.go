package main

import (
	"flag"
	"fmt"
	"gopkg.in/fsnotify.v1"
	"log"
	"os"
	"os/exec"
	"time"
)

var dirToWatch string
var showHelp bool
var abortOnError bool
var every int

func init() {
	flag.StringVar(&dirToWatch, "dir", ".", "The directory to monitor")
	flag.BoolVar(&showHelp, "help", false, "Show help file")
	flag.BoolVar(&showHelp, "abort", false, "Abort and stop monitoring when command fails")
	flag.IntVar(&every, "every", 1, "Run the command at most every given seconds")
}

func showUsage() {
	fmt.Println("Usage: " + os.Args[0] + " [options] <command> [args...]")
	fmt.Println("options:")
	flag.PrintDefaults()
}

func runCommand(args []string) error {
	var cmdargs []string
	if len(args) > 1 {
		cmdargs = args[1:len(args)]
	}
	cmd := exec.Command(args[0], cmdargs...)

	output, err := cmd.CombinedOutput()
	print(string(output))
	return err
}

func main() {
	flag.Parse()

	args := flag.Args()
	if showHelp || len(args) == 0 {
		showUsage()
		return
	}

	_, err := exec.LookPath(args[0])
	if err != nil {
		fmt.Printf("command not found: %s\n", args[0])
		return
	}

	runCommand(args)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	doRun := true
	done := make(chan bool)

	go func() {
		for range time.Tick(time.Duration(every) * time.Second) {
			if !doRun {
				continue
			}

			err := runCommand(args)
			doRun = false
			if err != nil && abortOnError {
				fmt.Printf("error: %o", err.Error())
				done <- true
			}
		}
	}()

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					doRun = true
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(dirToWatch)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
