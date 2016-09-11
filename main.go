package main

import (
	"flag"
	"fmt"
	"gopkg.in/fsnotify.v1"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

var dirToWatch string
var showHelp bool
var abortOnError bool
var every int

var eventFlags string

var includePattern string
var excludePattern string

func init() {
	flag.StringVar(&dirToWatch, "dir", ".", "The directory to monitor")
	flag.BoolVar(&showHelp, "help", false, "Show help file")
	flag.BoolVar(&showHelp, "abort", false, "Abort and stop monitoring when command fails")
	flag.IntVar(&every, "every", 1, "Run the command at most every given seconds")

	flag.StringVar(&eventFlags, "events", "0011",
		"Events to monitor: RENAME|DELETE|MODIFY|CREATE\n"+
			"\tProvide a bitstring with the indicated order\n"+
			"\tFor example, to monitor modify MODIFY events only, then set -events=0010\n"+
			"\tTo, monitor RENAME and DELETE events only, then set -events=1100\n\t")

	flag.StringVar(&includePattern, "include", ".*", "A regular expression of the files that will be monitored")
	flag.StringVar(&excludePattern, "exclude", "[^\\s\\S]", "A regular expression of the files that will be NOT monitored")
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

func parseBitstring(s string) (int, error) {
	n, err := strconv.ParseInt(s, 2, 8)
	return int(n), err
}

func main() {
	flag.Parse()

	args := flag.Args()
	if showHelp || len(args) == 0 {
		showUsage()
		return
	}

	// Check if command is a valid executable file
	_, err := exec.LookPath(args[0])
	if err != nil {
		fmt.Printf("command not found: %s\n", args[0])
		return
	}

	includeRx := regexp.MustCompile(includePattern)
	excludeRx := regexp.MustCompile(excludePattern)

	evflags, err := parseBitstring(eventFlags)
	if err != nil {
		fmt.Printf("error: invalid events, using 0011")
		evflags, _ = parseBitstring("0011")
	}

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
				if event.Op&fsnotify.Op(evflags) > 0 &&
					includeRx.MatchString(event.Name) &&
					!excludeRx.MatchString(event.Name) {

					log.Println("event:", event)
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
