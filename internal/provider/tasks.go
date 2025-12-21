package provider

import "github.com/ja-he/dayplan/internal/model"

type TaskLocationContext struct {
	Previous  *model.TaskID
	Next      *model.TaskID
	Parentage []model.TaskID
}

func (c TaskLocationContext) GetParentIDOrNil() *model.TaskID {
	if len(c.Parentage) > 0 {
		return &c.Parentage[len(c.Parentage)-1]
	}
	return nil
}

type TaskProvider interface {
	WithRoots(f func(roots []model.ReadableTask)) error
	WithTask(id model.TaskID, f func(t model.ReadableTask)) error
	WithTasks(ids []model.TaskID, f func(ts []model.ReadableTask)) error

	// returns the first child or, if given ID is nil, the first root task ID
	GetFirstChildTaskID(*model.TaskID) (*model.TaskID, error)
	// returns the last child or, if given ID is nil, the first root task ID
	GetLastChildTaskID(*model.TaskID) (*model.TaskID, error)

	GetLocationContext(id model.TaskID) (TaskLocationContext, error)
	GetCategory(id model.TaskID) (model.CategoryName, error)

	InsertFront(data model.ReadableTask, parentID *model.TaskID) (model.TaskID, error)
	InsertBack(data model.ReadableTask, parentID *model.TaskID) (model.TaskID, error)
	InsertBefore(data model.ReadableTask, anchorID model.TaskID) (model.TaskID, error)
	InsertAfter(data model.ReadableTask, anchorID model.TaskID) (model.TaskID, error)

	Remove(id model.TaskID) (model.ReadableTask, TaskLocationContext, error)

	Update(id model.TaskID, data model.ReadableTask) error

	Load() error
	Save() error
}
