package main

import (
	"container/heap"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

type Jobs struct {
	// The heap of jobs used as a priority queue
	List []*Job

	// Mapping for enabling efficient deletion
	CrontabMapping map[string]map[*Job]struct{}
}

var EmptyJobs = Jobs{[]*Job{}, map[string]map[*Job]struct{}{}}

func (js *Jobs) Marshal() (result [][]byte, err error) {
	result = make([][]byte, len(js.List))
	for i, j := range js.List {
		result[i], err = j.MarshalJSON()
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func NewJobsFromCrontabs(crontabs []Crontab) *Jobs {
	jobs := &Jobs{[]*Job{}, map[string]map[*Job]struct{}{}}

	now := time.Now()

	for _, crontab := range crontabs {
		jobs.Add(now, crontab)
	}
	return jobs
}

func (jobs *Jobs) Add(now time.Time, crontab Crontab) {
	for _, job := range crontab {
		job.nextRun = job.Next(now)
		heap.Push(jobs, job)
	}
}

func (jobs Jobs) Len() int { return len(jobs.List) }

func (jobs Jobs) Less(i, j int) bool {
	// Pop should give us the next one to be run
	return jobs.List[i].nextRun.Before(jobs.List[j].nextRun)
}

func (jobs *Jobs) Swap(i, j int) {
	jobs.List[i], jobs.List[j] = jobs.List[j], jobs.List[i]

	// Update indices
	jobs.List[i].index = i
	jobs.List[j].index = j

}

// Next job to be invoked
func (jobs Jobs) Top() *Job {
	return jobs.List[0]
}

func (jobs *Jobs) Push(x interface{}) {
	job := x.(*Job)

	if _, ok := jobs.CrontabMapping[job.crontab]; !ok {
		jobs.CrontabMapping[job.crontab] = map[*Job]struct{}{}
	}
	jobs.CrontabMapping[job.crontab][job] = struct{}{}

	job.index = len(jobs.List)
	jobs.List = append(jobs.List, job)
}

func (jobs *Jobs) Pop() interface{} {

	old := jobs.List
	n := len(old)
	job := old[n-1]
	job.index = -1 // for safety
	jobs.List = old[0 : n-1]

	delete(jobs.CrontabMapping[job.crontab], job)

	return job
}

// Return the next batch of jobs to run
func (jobs *Jobs) NextJobs() Jobs {
	if jobs.Len() == 0 {
		return EmptyJobs
	}

	runTime := jobs.Top().nextRun

	// Note, epislon can't be smaller than 1 second, I tried.
	const epsilon = 1 * time.Second
	runTimePlusEpsilon := runTime.Add(epsilon)

	// Pop all jobs due for this time
	jobsThisRun := Jobs{}
	for jobs.Top().nextRun == runTime {
		job := jobs.Top()
		job.nextRun = job.Next(runTimePlusEpsilon)
		// Reinsert at correct position for cost of ln(N)
		heap.Fix(jobs, 0)

		if job.nextRun.Equal(runTime) || job.nextRun.Before(runTime) {
			log.Fatalf("Logic error: next run time is wrong")
		}
		// log.Println("Job next:", job.nextRun)
		jobsThisRun.List = append(jobsThisRun.List, job)
	}

	return jobsThisRun
}

func (jobs *Jobs) Invoke(allJobs *Jobs) {
	// // Every job updating its own crontab. Hilarious.
	// for _, job := range jobs.List {
	// 	allJobs.UpdateCrontab(time.Now(), job.crontab)
	// }

	// return

	for _, job := range jobs.List {
		cmd := exec.Command("bash", "-c", fmt.Sprintf("echo %q", job.script))
		_ = cmd
		_ = os.Stdout
		_ = cmd.Run()
		// cmd.Stdout = os.Stdout
		// cmd.Stderr = os.Stderr
		// For now, we don't care about run errors
	}
}

func (jobs *Jobs) UpdateCrontab(when time.Time, crontab string) {

	// log.Printf("indices which need updating: %#+v", jobs.CrontabMapping[crontab])

	// Check in case it's not a crontab we've seen before
	if _, ok := jobs.CrontabMapping[crontab]; ok {
		// Remove each job from the heap, note index is maintained
		// by heap.Remove
		for job := range jobs.CrontabMapping[crontab] {
			if job.crontab != crontab {
				log.Panicf("Logic error, crontabs not equal: %v != %v ", job.crontab, crontab)
			}
			heap.Remove(jobs, job.index)
		}
	}

	c, err := ReadCrontab(crontab)
	if err != nil {
		log.Println("Error updating crontab: %v: %q", crontab, err)
		return
	}
	jobs.Add(when, c)
}
