package main

import (
	"testing"
)

// Ensure that whitespace makes no difference to splitting
func TestSplitWhitespaceInvariance(t *testing.T) {
	check := func(schedule, schedExpt, job, jobExpt string) {
		if schedule != schedExpt {
			t.Errorf("schedule != %q got %q", schedExpt, schedule)
		}
		if job != jobExpt {
			t.Errorf("job != %q got %q", jobExpt, job)
		}
	}

	var err error
	_ = err

	schedule, job, err := SplitCron("* * * * * hello world", false)
	check(schedule, "* * * * *", job, "hello world")

	schedule, job, err = SplitCron("* * * * *       hello world", false)
	check(schedule, "* * * * *", job, "hello world")

	schedule, job, err = SplitCron("  * *   * * *       hello world", false)
	check(schedule, "* * * * *", job, "hello world")

	schedule, job, err = SplitCron("* * * * * hello world", false)
	check(schedule, "* * * * *", job, "hello world")

	schedule, job, err = SplitCron("*/10 * * * * hello world", false)
	check(schedule, "*/10 * * * *", job, "hello world")

	schedule, job, err = SplitCron(`* * * * * foo@bar`, false)
	check(schedule, "* * * * *", job, "foo@bar")

	schedule, job, err = SplitCron(`00 01 * * * test`, false)
	check(schedule, "0 1 * * *", job, "test")
}

func TestParseCronLine(t *testing.T) {
	env := map[string]string{}
	ParseCronLine(env, "TestParseCronLine", "# hello world")
	ParseCronLine(env, "TestParseCronLine", "@midnight hello world")
	ParseCronLine(env, "TestParseCronLine", "ASSIGN=a/value")

}
