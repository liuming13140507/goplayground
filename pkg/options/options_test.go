package options

import (
	"testing"
)

func TestTaskManagerOptions(t *testing.T) {
	t.Run("Default configuration", func(t *testing.T) {
		tm, err := NewTaskManager()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if tm.Name() != defaultName {
			t.Errorf("expected name %s, got %s", defaultName, tm.Name())
		}
		if tm.TaskType() != defaultTaskType {
			t.Errorf("expected type %d, got %d", defaultTaskType, tm.TaskType())
		}
	})

	t.Run("With custom options", func(t *testing.T) {
		list := []int32{1, 2, 3}
		tm, err := NewTaskManager(
			WithName("my-task"),
			WithTaskType(100),
			WithTaskList(list),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if tm.Name() != "my-task" {
			t.Errorf("expected name my-task, got %s", tm.Name())
		}
		if tm.TaskType() != 100 {
			t.Errorf("expected type 100, got %d", tm.TaskType())
		}
		if len(tm.TaskList()) != 3 {
			t.Errorf("expected list length 3, got %d", len(tm.TaskList()))
		}
	})

	t.Run("Validation error", func(t *testing.T) {
		_, err := NewTaskManager(WithName(""))
		if err == nil {
			t.Error("expected error for empty name, got nil")
		}
	})
}
