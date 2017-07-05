package tasks

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/utils"
)

// Task configures tasks that run periodically
type Task struct {
	Name         string      `mapstructure:"name"`
	Command      interface{} `mapstructure:"command"`
	Frequency    string      `mapstructure:"frequency"`
	Timeout      string      `mapstructure:"timeout"`
	freqDuration time.Duration
	cmd          *commands.Command
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

	if task.Command == nil {
		return fmt.Errorf("Task did not provide a command")
	}

	freq, err := utils.ParseDuration(task.Frequency)
	if err != nil {
		return fmt.Errorf("Unable to parse frequency %s: %v", task.Frequency, err)
	}
	if freq < time.Millisecond {
		return fmt.Errorf("Frequency %v cannot be less that %v", freq, taskMinDuration)
	}
	task.freqDuration = freq

	if task.Timeout == "" {
		task.Timeout = task.Frequency
	}
	cmd, err := commands.NewCommand(task.Command, task.Timeout,
		log.Fields{"process": "task", "task": task.Name})
	if cmd.TimeoutDuration < taskMinDuration {
		return fmt.Errorf("Timeout %v cannot be less that %v", cmd.TimeoutDuration, taskMinDuration)
	}
	cmd.Name = fmt.Sprintf("task[%s]", task.Name)
	task.cmd = cmd

	return nil
}

// PollTime returns the frequency of the task
func (t *Task) PollTime() time.Duration {
	return t.freqDuration
}

// PollStop kills a running task
func (t *Task) PollStop() {
	log.Debugf("task[%s].PollStop", t.Name)
	if t.cmd != nil {
		t.cmd.Kill()
	}
}

// PollAction runs the task
func (t *Task) PollAction() {
	commands.RunWithTimeout(t.cmd)
}
