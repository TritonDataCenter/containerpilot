package tasks

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/utils"
)

// TaskConfig configures tasks that run periodically
type TaskConfig struct {
	Name      string   `json:"name"`
	Args      []string `json:"command"`
	Frequency string   `json:"frequency"`
	Timeout   string   `json:"timeout,omitempty"`

	freqDuration    time.Duration
	timeoutDuration time.Duration
	cmd             *exec.Cmd
	quitChannel     chan int
	ticker          *time.Ticker
	logWriters      []io.WriteCloser
}

// Task a task that runs periodically until killed
type Task interface {
	Start() error
	Stop()
}

func (t *TaskConfig) closeLogs() {
	if t.logWriters == nil {
		return
	}
	for _, w := range t.logWriters {
		if err := w.Close(); err != nil {
			log.Errorf("Unable to close log writer : %v", err)
		}
	}
}

// NewTasks parses json config into an array of Tasks
func NewTasks(raw json.RawMessage) ([]Task, error) {
	var tasks []Task
	if raw == nil {
		return tasks, nil
	}
	var configs []*TaskConfig
	if err := json.Unmarshal(raw, &configs); err != nil {
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

func parseTask(task *TaskConfig) error {
	if task.Args == nil || len(task.Args) == 0 {
		return fmt.Errorf("Task did not provide a command")
	}
	if task.Name == "" {
		task.Name = strings.Join(task.Args, " ")
	}
	freq, err := time.ParseDuration(task.Frequency)
	if err != nil {
		return fmt.Errorf("Unable to parse frequency %s: %v", task.Frequency, err)
	}
	task.freqDuration = freq
	task.timeoutDuration = freq
	if task.Timeout != "" {
		timeout, err := time.ParseDuration(task.Timeout)
		if err != nil {
			return fmt.Errorf("Unable to parse timeout %s: %v", task.Timeout, err)
		}
		task.timeoutDuration = timeout
	} else {
		task.Timeout = task.Frequency
	}
	task.quitChannel = make(chan int)
	return nil
}

// Start starts the task
func (t *TaskConfig) Start() error {

	t.ticker = time.NewTicker(t.freqDuration)
	quit := make(chan int)

	// Accept quit signal
	go func() {
		select {
		case <-t.quitChannel:
			log.Debugf("task[%s].Start#gofunc <-quitChannel", t.Name)
			t.ticker.Stop()
			quit <- 0
			t.kill(t.cmd)
			return
		}
	}()

	// Run command
	go func() {
		defer t.ticker.Stop()
		for {
			select {
			case <-t.ticker.C:
				log.Debugf("task[%s].Start#gofunc <-t.ticker.C", t.Name)
				t.execute()
			case <-quit:
				log.Debugf("task[%s].Start#gofunc <-quit", t.Name)
				return
			}
		}
	}()
	return nil
}

func (t *TaskConfig) kill(cmd *exec.Cmd) error {
	log.Debugf("task[%s].kill", t.Name)
	if cmd != nil && cmd.Process != nil {
		log.Warnf("Killing task %s: %d", t.Name, cmd.Process.Pid)
		return cmd.Process.Kill()
	}
	return nil
}

func (t *TaskConfig) execute() {
	cmd := utils.ArgsToCmd(t.Args)
	t.cmd = cmd
	fields := make(map[string]interface{})
	fields["task"] = t.Name
	t.logWriters = []io.WriteCloser{
		utils.NewLogWriter(fields, 5),
		utils.NewLogWriter(fields, 4),
	}
	cmd.Stdout = t.logWriters[0]
	cmd.Stderr = t.logWriters[1]
	defer t.closeLogs()
	if err := cmd.Start(); err != nil {
		log.Errorf("Unable to start task %s: %v", t.Name, err)
		return
	}
	t.run(cmd)
	log.Debugf("task[%s].execute complete", t.Name)
}

func (t *TaskConfig) run(cmd *exec.Cmd) {
	ticker := time.NewTicker(t.timeoutDuration)
	quit := make(chan int)
	go func() {
		defer ticker.Stop()
		select {
		case <-ticker.C:
			log.Warnf("Task %s timeout after %s: '%s'", t.Name, t.Timeout, t.Args)
			if err := t.kill(cmd); err != nil {
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

// Stop stops a task from executing and kills the currently running execution
func (t *TaskConfig) Stop() {
	log.Debugf("task[%s].Stop", t.Name)
	t.quitChannel <- 0
	log.Debugf("task[%s].Stop quit", t.Name)
}
