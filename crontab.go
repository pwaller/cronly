package main

import (
	"bufio"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/gorhill/cronexpr"
)

type Job struct {
	// Schedule
	*cronexpr.Expression

	environment map[string]string

	script  string // Script to run
	crontab string // Crontab originated from

	index int // position in queue

	nextRun time.Time
}
type Crontab []*Job

var ErrNotEnoughFields = errors.New("not enough fields")

// Note: this might be better as a regexp...
// shameless rip of strings.FieldsFunc, with the exception
// that this supports splitting a limited number of fields
func FieldsN(s string, max int) ([]string, error) {

	a := []string{}
	fieldStart := -1 // Set to -1 when looking for start of field.
	last := -1
	for i, character := range s {
		if unicode.IsSpace(character) {
			if fieldStart >= 0 {
				// The start of a new section of whitespace,
				// add the string just seen to `a`.
				a = append(a, s[fieldStart:i])
				fieldStart = -1
			}
		} else if fieldStart == -1 {
			// The first non-isSpace character after an IsSpace
			if len(a) >= max-1 {
				last = i
				break
			}
			fieldStart = i
		}
	}
	if last == -1 {
		return a, ErrNotEnoughFields
	}
	a = append(a, s[last:])
	return a, nil
}

// Given "* * * * * foo bar" return ["* * * * *", "foo bar"]
func SplitCron(input string, at bool) (schedule, exec string, err error) {
	nfields := 6
	if at {
		nfields = 2
	}

	fields, err := FieldsN(input, nfields)
	if err != nil {
		return
	}
	schedule = strings.Join(fields[:nfields-1], " ")
	exec = fields[nfields-1]

	schedule = RemoveLeadingZeroes(schedule)

	return
}

// Needed because cronexpr can't cope with them
func RemoveLeadingZeroes(input string) string {
	replace := func(in string) string {
		result := strings.TrimLeft(in, "0")
		if result == "" {
			return "0"
		}
		return result
	}
	return leading_zeros.ReplaceAllStringFunc(input, replace)
}

var (
	assignment       = regexp.MustCompile(`^([a-zA-Z0-9_]+)\s*=\s*(.*)$`)
	starts_with_at   = regexp.MustCompile(`^\s*@`)
	starts_with_hash = regexp.MustCompile(`^\s*#`)
	blank_line       = regexp.MustCompile(`^\s*$`)
	leading_zeros    = regexp.MustCompile(`\b0+[0-9]`)
)

func ParseCronLine(env map[string]string, filename, line string) (job *Job, err error) {

	if starts_with_hash.MatchString(line) || blank_line.MatchString(line) {
		return

	} else if assignment.MatchString(line) {
		// Environment assignment
		matches := assignment.FindStringSubmatch(line)[1:]
		left, right := matches[0], matches[1]
		env[left] = right

	} else {
		// Schedule line
		has_at := starts_with_at.MatchString(line)

		var scheduleText, script string
		scheduleText, script, err = SplitCron(line, has_at)
		if err != nil {
			return
		}

		schedule, err := cronexpr.Parse(scheduleText)
		if err != nil {
			// log.Printf("Fields: %q -- %q -- %q", scheduleText, script, line)
			return nil, err
		}

		// Initialize job with zero time
		job = &Job{schedule, env, script, filename, -1, time.Time{}}
	}

	return
}

func ParseCron(filename string, input io.Reader) Crontab {
	result := []*Job{}

	env := map[string]string{}

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		job, err := ParseCronLine(env, filename, line)
		if job != nil {
			result = append(result, job)
		}
		if err != nil {
			log.Println("Error parsing line", filename, err, line)
		}
	}
	return result
}

func ReadCrontab(cronpath string) (Crontab, error) {
	fd, err := os.Open(cronpath)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	return ParseCron(cronpath, fd), nil
}

func ReadCrontabs(crondir string) []Crontab {
	crontabs, err := ioutil.ReadDir(crondir)
	if err != nil {
		panic(err)
	}

	result := []Crontab{}
	for _, crontab := range crontabs {
		c, err := ReadCrontab(path.Join(crondir, crontab.Name()))
		if err != nil {
			log.Println("Error parsing crontab %v: %q", crontab, err)
			continue
		}
		result = append(result, c)
	}
	return result
}
