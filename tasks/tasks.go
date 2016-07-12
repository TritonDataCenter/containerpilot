package tasks

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/utils"
)

// Task configures tasks that run periodically
type Task struct {
	Name      string      `mapstructure:"name"`
	Command   interface{} `mapstructure:"command"`
	Frequency string      `mapstructure:"frequency"`
	Timeout   string      `mapstructure:"timeout"`

	Args            []string
	freqDuration    time.Duration
	timeoutDuration time.Duration
	cmd             *exec.Cmd
	ticker          *time.Ticker
	logWriters      []io.WriteCloser
}

var taskMinDuration = 1 * time.Millisecond

// NewTasks parses json config into an array of Tasks
func NewTasks(raw []interface{}) ([]*Task, error) {
	var tasks []*Task
	if raw == nil {
		return tasks, nil
	}
	var configs []*Task
	if err := utils.DecodeRaw(raw, &configs); err != nil {
		return nil, fmt.Errorf("Task configuration error: %v", err)
	}
	for _, t := range configs {
		if err := parseTask(t); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func parseTask(task *Task) error {
	args, err := utils.ToStringArray(task.Command)
	if err != nil {
		return err
	}
	task.Args = args
	if task.Args == nil || len(task.Args) == 0 {
		return fmt.Errorf("Task did not provide a command")
	}
	if task.Name == "" {
		task.Name = strings.Join(task.Args, " ")
	}
	freq, err := utils.ParseDuration(task.Frequency)
	if err != nil {
		return fmt.Errorf("Unable to parse frequency %s: %v", task.Frequency, err)
	}
	if freq < time.Millisecond {
		return fmt.Errorf("Frequency %v cannot be less that %v", freq, taskMinDuration)
	}
	task.freqDuration = freq
	task.timeoutDuration = freq
	if task.Timeout != "" {
		timeout, err := utils.ParseDuration(task.Timeout)
		if err != nil {
			return fmt.Errorf("Unable to parse timeout %s: %v", task.Timeout, err)
		}
		if timeout < taskMinDuration {
			return fmt.Errorf("Timeout %v cannot be less that %v", timeout, taskMinDuration)
		}
		task.timeoutDuration = timeout
	} else {
		task.Timeout = task.Frequency
	}
	return nil
}

// PollTime returns the frequency of the task
func (t *Task) PollTime() time.Duration {
	return t.freqDuration
}

// PollStop kills a running task
func (t *Task) PollStop() {
	log.Debugf("task[%s].PollStop", t.Name)
	t.kill()
}

// PollAction runs the task
func (t *Task) PollAction() {
	log.Debugf("task[%s].PollAction", t.Name)
	cmd := commands.ArgsToCmd(t.Args)
	t.cmd = cmd
	fields := log.Fields{"process": "task", "task": t.Name}
	stdout := utils.NewLogWriter(fields, log.InfoLevel)
	stderr := utils.NewLogWriter(fields, log.DebugLevel)
	t.logWriters = []io.WriteCloser{stdout, stderr}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	defer t.closeLogs()
	log.Debugf("task[%s].PollAction start", t.Name)
	if err := cmd.Start(); err != nil {
		log.Errorf("Unable to start task %s: %v", t.Name, err)
		return
	}
	t.run(cmd)
	log.Debugf("task[%s].PollAction complete", t.Name)
}

func (t *Task) kill() error {
	log.Debugf("task[%s].kill", t.Name)
	if t.cmd != nil && t.cmd.Process != nil {
		log.Warnf("Killing task %s: %d", t.Name, t.cmd.Process.Pid)
		return t.cmd.Process.Kill()
	}
	return nil
}

func (t *Task) run(cmd *exec.Cmd) {
	ticker := time.NewTicker(t.timeoutDuration)
	quit := make(chan int)
	go func() {
		defer ticker.Stop()
		select {
		case <-ticker.C:
			log.Warnf("Task %s timeout after %s: '%s'", t.Name, t.Timeout, t.Args)
			if err := t.kill(); err != nil {
				log.Errorf("Error killing task %s: %v", t.Name, err)
			}
			// Swallow quit signal
			log.Debugf("task[%s].run#gofunc swallow quit", t.Name)
			<-quit
			log.Debugf("task[%s].run#gofunc swallow quit complete", t.Name)
			return
		case <-quit:
			log.Debugf("task[%s].run#gofunc received quit", t.Name)
			return
		}
	}()
	log.Debugf("task[%s].run waiting for PID %d: ", t.Name, cmd.Process.Pid)
	_, err := cmd.Process.Wait()
	if err != nil {
		log.Errorf("Task %s exited with error: %v", t.Name, err)
	}
	log.Debugf("task[%s].run sent timeout quit", t.Name)
	quit <- 0
	log.Debugf("task[%s].run complete", t.Name)
}

func (t *Task) closeLogs() {
	if t.logWriters == nil {
		return
	}
	for _, w := range t.logWriters {
		if err := w.Close(); err != nil {
			log.Errorf("Unable to close log writer : %v", err)
		}
	}
}
