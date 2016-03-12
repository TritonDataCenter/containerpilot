package tasks

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
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
	lock            *sync.Mutex
	ticker          *time.Ticker
	logWriters      []io.WriteCloser
}

// Task a task that runs periodically until killed
type Task interface {
	Start() error
	Stop() error
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
	task.lock = &sync.Mutex{}
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

	go func() {
		for {
			select {
			case <-t.ticker.C:
				t.execute()
			case <-t.quitChannel:
				t.ticker.Stop()
				return
			}
		}
	}()
	return nil
}

func (t *TaskConfig) kill(cmd *exec.Cmd) error {
	if cmd != nil && cmd.Process != nil {
		log.Warnf("Killing task %s: %d", t.Name, cmd.Process.Pid)
		return cmd.Process.Kill()
	}
	return nil
}

func (t *TaskConfig) execute() {
	t.lock.Lock()
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
	if err := cmd.Start(); err != nil {
		defer t.lock.Unlock()
		defer t.closeLogs()
		log.Errorf("Unable to start task %s: %v", t.Name, err)
		return
	}
	t.lock.Unlock()
	t.run(cmd)
}

func (t *TaskConfig) run(cmd *exec.Cmd) {
	defer t.closeLogs()
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
			<-quit
			return
		case <-quit:
			return
		}
	}()
	_, err := cmd.Process.Wait()
	if err != nil {
		log.Errorf("Task %s exited with error: %v", t.Name, err)
	}
	quit <- 0
}

// Stop stops a task from executing and kills the currently running execution
func (t *TaskConfig) Stop() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	log.Infof("task.Stop: %s", t.Name)
	t.quitChannel <- 0
	return t.kill(t.cmd)
}
