package options

import (
	"fmt"
)

// TaskManager is the main struct.
// Industry standard: Export the struct, but keep configuration-related fields unexported
// if you want to ensure they are only set during initialization via Options.
type TaskManager struct {
	name     string
	taskType int32
	taskList []int32
}

// config holds all configuration for TaskManager.
// Industry standard: Separate config from the main struct to keep the main struct clean.
type config struct {
	name     string
	taskType int32
	taskList []int32
}

// Option defines the functional option type.
type Option func(*config) error

// Default values constant/variable.
const (
	defaultName     = "default-task"
	defaultTaskType = 1
)

// WithName sets the name.
// Industry standard: Return an error if validation is needed.
func WithName(name string) Option {
	return func(c *config) error {
		if name == "" {
			return fmt.Errorf("name cannot be empty")
		}
		c.name = name
		return nil
	}
}

// WithTaskType sets the task type.
func WithTaskType(t int32) Option {
	return func(c *config) error {
		c.taskType = t
		return nil
	}
}

// WithTaskList sets the task list.
func WithTaskList(list []int32) Option {
	return func(c *config) error {
		// Defensive copy to prevent external modification
		c.taskList = make([]int32, len(list))
		copy(c.taskList, list)
		return nil
	}
}

// NewTaskManager creates a new instance with functional options.
// Industry standard: Return (obj, error) if options can fail.
func NewTaskManager(opts ...Option) (*TaskManager, error) {
	// 1. Initialize with default config
	c := &config{
		name:     defaultName,
		taskType: defaultTaskType,
		taskList: make([]int32, 0),
	}

	// 2. Apply all options
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// 3. Assemble the final object
	return &TaskManager{
		name:     c.name,
		taskType: c.taskType,
		taskList: c.taskList,
	}, nil
}

// Getters (since fields are unexported)
func (t *TaskManager) Name() string      { return t.name }
func (t *TaskManager) TaskType() int32   { return t.taskType }
func (t *TaskManager) TaskList() []int32 { return t.taskList }
