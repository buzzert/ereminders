package main

import (
	"bufio"
	"errors"
	"log"
	"os"
	"path/filepath"
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

func parseJobFile(filePath string) (*MailJob, error) {
	/*
	 * Syntax for job file:
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
			res, err := dateParser.Parse(scanner.Text(), time.Now())
			if err != nil || res == nil {
				return nil, err
			}

			job.Date = res.Time
		case EmptyLine:
			break
		case MessageLines:
			fallthrough
		default:
			job.Message += scanner.Text()
		}

		lineno++
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
		if job.Date.Sub(now) <= 0 {
			// The job date has already passed. Execute.
			err := job.Execute()
			if err != nil {
				log.Fatal("Error executing job: ", err)
			} else {
				// Self destruct file
				job.SelfDestruct()
			}
		} else {
			if nextJobIndex == -1 || jobs[nextJobIndex].Date.Sub(job.Date) > 0 {
				nextJobIndex = idx
			}

			newJobList = append(newJobList, job)
		}
	}

	var timeUntilNextJob time.Duration = -1
	if nextJobIndex != -1 {
		earliestNextJob := jobs[nextJobIndex]
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

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: ", os.Args[0], " jobsPath")
	}

	jobsPath := os.Args[1]
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

	jobs := parseJobList(files)

	// Spawn off filesystem monitor
	go startMonitoringFilesystem(jobsPath, addedJobChannel, removedJobNameChannel)

	for {
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

		// Or wait until the next job
		case <-time.After(timeUntilNextJob):
			continue
		}
	}
}
