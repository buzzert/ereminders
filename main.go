package main

import (
	"bufio"
	"errors"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/common"
	"github.com/olebedev/when/rules/en"
)

var dateParser = when.New(nil)

type safeJobList struct {
	list []MailJob
	mux  sync.Mutex
}

func nextDateForRepeatType(repeatType RepeatType, date time.Time) time.Time {
	switch repeatType {
	case RepeatDaily:
		date = date.AddDate(0, 0, 1)
	case RepeatWeekly:
		date = date.AddDate(0, 0, 7)
	case RepeatMonthly:
		date = date.AddDate(0, 1, 0)
	}

	return date
}

func parseJobFile(filePath string) (*MailJob, error) {
	/*
	 * Syntax for job file:
	 * 0. <optional> repeat {daily, weekly, monthly}
	 * 1. Date in natural language format
	 *    (relative times start with "in")
	 * 2. <empty line>
	 * 3. Email message contents
	 */

	type JobFileLine int
	const (
		DateLine JobFileLine = iota
		EmptyLine
		MessageLines
	)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	var job MailJob
	job.FilePath = filePath

	var lineno JobFileLine
	for scanner.Scan() {
		switch lineno {
		case DateLine:
			dateStr := scanner.Text()

			splitDateStr := strings.Split(dateStr, " ")
			if splitDateStr[0] == "repeat" {
				if len(splitDateStr) != 2 {
					return nil, errors.New("Error parsing repeat date")
				}

				frequency := strings.ToLower(splitDateStr[1])
				switch frequency {
				case "daily":
					job.Repeat = RepeatDaily
				case "weekly":
					job.Repeat = RepeatWeekly
				case "monthly":
					job.Repeat = RepeatMonthly
				}
			} else {
				res, err := dateParser.Parse(dateStr, time.Now())
				if err != nil || res == nil {
					return nil, err
				}

				job.Date = nextDateForRepeatType(job.Repeat, res.Time)
				lineno++
			}

		case EmptyLine:
			lineno++
		case MessageLines:
			fallthrough
		default:
			job.Message += scanner.Text()
			lineno++
		}
	}

	if lineno < MessageLines {
		return nil, errors.New("Invalid file format")
	}

	return &job, nil
}

func parseJobList(jobPaths []string) []MailJob {
	var jobs []MailJob

	for _, jobStr := range jobPaths {
		if !isValidJobFile(jobStr) {
			continue
		}

		log.Println("Parsing job file: ", jobStr)
		job, err := parseJobFile(jobStr)
		if job == nil || err != nil {
			log.Println("Error parsing job ", jobStr, ": ", err)

			// Move the file to signal that there was an error parsing it
			os.Rename(jobStr, jobStr+".ERROR")
			continue
		}

		jobs = append(jobs, *job)
	}

	return jobs
}

// executeJobList executes the list of jobs and returns the remaining jobs and the
// duration until the next job
func executeJobList(jobs []MailJob) ([]MailJob, time.Duration) {
	var newJobList []MailJob

	now := time.Now()

	nextJobIndex := -1
	for idx, job := range jobs {
		jobRemains := true

		if job.Date.Sub(now) <= 0 {
			println("Executing job: " + job.String())

			// The job date has already passed. Execute.
			err := job.Execute()
			if err != nil {
				log.Fatal("Error executing job: ", err)
			} else {
				if job.Repeat == RepeatNone {
					// Self destruct file
					job.SelfDestruct()
					jobRemains = false
				} else {
					job.Date = nextDateForRepeatType(job.Repeat, job.Date)
				}
			}
		}

		if jobRemains {
			if nextJobIndex == -1 || newJobList[nextJobIndex].Date.Sub(job.Date) > 0 {
				nextJobIndex = idx
			}

			newJobList = append(newJobList, job)
		}
	}

	var timeUntilNextJob time.Duration = -1
	if nextJobIndex != -1 {
		earliestNextJob := newJobList[nextJobIndex]
		timeUntilNextJob = earliestNextJob.Date.Sub(now)
	}

	return newJobList, timeUntilNextJob
}

func isValidJobFile(path string) bool {
	basename := filepath.Base(path)

	// Ignmore files that start with .
	if basename[0] == '.' {
		return false
	}

	// Ignore files that had a parse error
	if filepath.Ext(path) == ".ERROR" {
		return false
	}

	return true
}

func startListeningForCommands(commandChannel chan string) {
	log.Println("Daemon is listening for commands")

	for {
		commandChannel <- ReadCommand()
	}
}

func startMonitoringFilesystem(watchPath string, addedJobChannel chan MailJob, removedJobNameChannel chan string) {
	log.Println("Beginning fs monitoring")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	watcher.Add(watchPath)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			log.Println("Got FS event! ", event)
			if !isValidJobFile(event.Name) {
				continue
			}

			if event.Op&fsnotify.Create == fsnotify.Create {
				log.Println("FSEvent: Found new job")

				newJob, _ := parseJobFile(event.Name)
				if newJob != nil {
					addedJobChannel <- *newJob
				}
			}

			if event.Op&fsnotify.Remove == fsnotify.Remove {
				log.Println("FSEvent: Removing job")
				removedJobNameChannel <- event.Name
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("FS event error: ", err)
		}
	}
}

func jobListDescription(jobList []MailJob) string {
	now := time.Now()

	var description string
	for _, job := range jobList {
		timeRemaining := job.Date.Sub(now)
		description += "in " + timeRemaining.String() + ": "
		description += job.String()
		description += "\n"
	}

	return description
}

func startDaemon(jobsPath string) {
	if len(jobsPath) == 0 {
		println("Usage: ereminders {-d [reminders directory] | -l}")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if _, err := os.Stat(jobsPath); os.IsNotExist(err) {
		log.Fatal("Error: jobsPath does not exist")
	}

	dateParser.Add(en.All...) // add english locale
	dateParser.Add(common.All...)

	files, err := filepath.Glob(jobsPath + "/*")
	if err != nil {
		log.Fatal("Error enumerating jobs path: ", err)
	}

	var timeUntilNextJob time.Duration = -1

	addedJobChannel := make(chan MailJob)
	removedJobNameChannel := make(chan string)
	commandChannel := make(chan string)

	jobs := parseJobList(files)

	// Spawn off filesystem monitor
	go startListeningForCommands(commandChannel)
	go startMonitoringFilesystem(jobsPath, addedJobChannel, removedJobNameChannel)

	running := true
	for running {
		jobs, timeUntilNextJob = executeJobList(jobs)

		if timeUntilNextJob == -1 {
			// Basically wait until infinity until fsevent occurs.
			timeUntilNextJob = time.Duration(time.Hour * 999)
		}

		log.Println("Time until next job: ", timeUntilNextJob, ". Zzzzzzz...")
		select {
		// Signal from the fs monitor -- added jobs
		case addedJob := <-addedJobChannel:
			jobs = append(jobs, addedJob)
			continue

		// Removed jobs
		case removedJobName := <-removedJobNameChannel:
			for i, job := range jobs {
				if job.FilePath == removedJobName {
					// Remove job from list
					jobs = append(jobs[:i], jobs[i+1:]...)
					break
				}
			}
			continue

		case commandString := <-commandChannel:
			println("got command")
			if commandString == "exit" {
				running = false
			} else if commandString == "list" {
				result := "Time until next job: " + timeUntilNextJob.String() + "\n"
				result += jobListDescription(jobs)
				SendResponse(result)
			} else {
				SendResponse("unrecognized")
			}
			continue

		// Or wait until the next job
		case <-time.After(timeUntilNextJob):
			continue
		}
	}
}

func main() {
	var showList bool
	var jobsPath string

	daemonUsage := "Run as daemon and watch provided directory"
	flag.StringVar(&jobsPath, "d", "", daemonUsage)

	listUsage := "List all currently scheduled jobs"
	flag.BoolVar(&showList, "l", false, listUsage)

	flag.Parse()

	if showList {
		response := TransmitCommand("list")
		println(response)
		os.Exit(1)
	}

	startDaemon(jobsPath)
}
