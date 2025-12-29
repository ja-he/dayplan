package backend

import (
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/provider"
)

type BacklogYamlIoProvider struct {
	tasks []*model.Task
	mtx   sync.RWMutex

	filePath string

	dirty bool

	log zerolog.Logger
}

func NewBacklogYamlIoProvider(filePath string) (*BacklogYamlIoProvider, error) {
	b := &BacklogYamlIoProvider{
		log:      log.With().Str("component", "backlog-yaml-provider").Logger(),
		filePath: filePath,
	}
	return b, nil
}

// backlogFromReader reads and deserializes a backlog from the io.Reader and returns the
// backlog.
func (b *BacklogYamlIoProvider) loadFromReaderUnsafe(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("unable to read from reader (%s)", err.Error())
	}

	stored := BacklogStored{}
	err = yaml.Unmarshal(data, &stored)
	if err != nil {
		return fmt.Errorf("yaml unmarshaling error (%s)", err.Error())
	}
	log.Debug().Int("N-Cats", len(stored.TasksByCategory)).Msg("read storeds")

	referencedTasks := make(map[model.TaskID]struct{})
	alreadyFound := func(id model.TaskID) bool {
		_, ok := referencedTasks[id]
		return ok
	}
	markFound := func(id model.TaskID) {
		referencedTasks[id] = struct{}{}
	}
	var findTaskByCatAndName func(s []*model.Task, cat model.CategoryName, name string, parentsOfTask []string, currentQueryPathParents []string) *model.TaskID
	findTaskByCatAndName = func(s []*model.Task, cat model.CategoryName, name string, parentsOfTask []string, currentQueryPathParents []string) *model.TaskID {
		b.log.Trace().Msgf("Will try to find task '%s':'%s' in %v", cat, name, s)
		if len(parentsOfTask) == len(currentQueryPathParents) {
			log.Debug().Msgf("Considering comps in %v.", s)
			index := slices.IndexFunc(s, func(e *model.Task) bool { return e.Category == cat && e.Name == name })
			if index != -1 && !alreadyFound(s[index].ID) {
				markFound(s[index].ID)
				return &s[index].ID
			}
		} else {
			log.Debug().Msgf("Not considering comp in %v due to len diff %d != %d.", s, len(parentsOfTask), len(currentQueryPathParents))
		}
		for _, t := range s {
			foundID := findTaskByCatAndName(t.Subtasks, cat, name, parentsOfTask, append(currentQueryPathParents, t.Name))
			if foundID != nil {
				return foundID
			}
		}
		return nil
	}
	var mapSubtasks func(cat model.CategoryName, tasks []BaseTask, parentsOfTaskNames []string) []*model.Task
	toTask := func(cat model.CategoryName, bt BaseTask, parentsOfTaskNames []string) *model.Task {
		foundTaskID := findTaskByCatAndName(b.tasks, cat, bt.Name, parentsOfTaskNames, nil)
		var id model.TaskID
		if foundTaskID != nil {
			id = *foundTaskID
		} else {
			log.Debug().Msgf("Could not relate task '%s':'%s' to existing.", cat, bt.Name)
			id = model.TaskID(uuid.NewString())
		}

		return &model.Task{
			ID:       id,
			Name:     bt.Name,
			Category: cat,
			Duration: bt.Duration,
			Deadline: bt.Deadline,
			Subtasks: mapSubtasks(cat, bt.Subtasks, append(parentsOfTaskNames, bt.Name)),
		}
	}
	mapSubtasks = func(cat model.CategoryName, tasks []BaseTask, parentsOfTaskNames []string) []*model.Task {
		result := []*model.Task{}
		for _, t := range tasks {
			result = append(result, toTask(cat, t, parentsOfTaskNames))
		}
		return result
	}

	var newTasks []*model.Task
	for cat, tasks := range stored.TasksByCategory {
		catName := model.CategoryName(cat)
		for _, task := range tasks {
			newTasks = append(newTasks, toTask(catName, task, nil))
		}
	}
	// sort to ensure more consistent ordering
	sort.Sort(model.TasksByDeadline(newTasks))

	b.tasks = newTasks

	return nil
}

type BacklogStored struct {
	TasksByCategory map[string][]BaseTask `yaml:",inline"`
}
type BaseTask struct {
	Name     string         `yaml:"name"`
	Duration *time.Duration `yaml:"duration,omitempty"`
	Deadline *time.Time     `yaml:"deadline,omitempty"`
	Subtasks []BaseTask     `yaml:"subtasks,omitempty"`
}

func createBaseTaskFromTask(t *model.Task) BaseTask {
	if t == nil {
		return BaseTask{}
	}
	result := BaseTask{
		Name:     t.Name,
		Duration: t.Duration,
		Deadline: t.Deadline,
		Subtasks: make([]BaseTask, 0, len(t.Subtasks)),
	}
	for _, subtask := range t.Subtasks {
		if t.Category != subtask.Category {
			log.Warn().
				Str("subtask", subtask.Name).
				Str("parent-task", t.Name).
				Str("subtask-category", string(subtask.Category)).
				Str("parent-task-category", string(t.Category)).
				Msg("subtask has different category from parent, which will be lost")
		}
		result.Subtasks = append(result.Subtasks, createBaseTaskFromTask(subtask))
	}
	return result
}

// ----------------------

func ptrsToReadables(m *[]*model.Task) *[]model.ReadableTask {
	if m == nil {
		return nil
	}
	result := make([]model.ReadableTask, len(*m))
	for i, t := range *m {
		result[i] = t
	}
	return &result
}
func convertReadablesToOwnedSlice(r []model.ReadableTask) []*model.Task {
	var result []*model.Task
	for _, t := range r {
		result = append(result, taskFromReadable(t))
	}
	return result
}
func taskFromReadable(t model.ReadableTask) *model.Task {
	return &model.Task{
		ID:       model.TaskID(uuid.NewString()),
		Name:     t.GetName(),
		Category: t.GetCategory(),
		Duration: t.GetDuration(),
		Deadline: t.GetDeadline(),
		Subtasks: convertReadablesToOwnedSlice(t.GetSubtasks()),
	}
}

func (b *BacklogYamlIoProvider) WithRoots(f func(roots []model.ReadableTask)) error {
	b.mtx.RLock()
	defer b.mtx.RUnlock()
	f(*ptrsToReadables(&b.tasks))
	return nil
}
func (b *BacklogYamlIoProvider) WithTask(id model.TaskID, f func(t model.ReadableTask)) error {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	t, _, _, err := b.locateUnsafe(id)
	if err != nil {
		return fmt.Errorf("Unable to locate task '%s' (%w)", id, err)
	}
	if t == nil {
		return fmt.Errorf("Unable to locate task '%s' (but no error!)", id)
	}

	f(t)

	return nil
}
func (b *BacklogYamlIoProvider) WithTasks(ids []model.TaskID, f func(ts []model.ReadableTask)) error {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	var tasks []model.ReadableTask
	for _, id := range ids {
		t, _, _, err := b.locateUnsafe(id)
		if err != nil {
			return fmt.Errorf("Unable to locate task '%s' (%w)", id, err)
		}
		if t == nil {
			return fmt.Errorf("Unable to locate task '%s' (but no error!)", id)
		}
		tasks = append(tasks, t)
	}

	f(tasks)
	return nil
}

func (b *BacklogYamlIoProvider) GetFirstChildTaskID(id *model.TaskID) (*model.TaskID, error) {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	if id == nil {
		if len(b.tasks) > 0 {
			firstChildTaskID := b.tasks[0].ID
			return &firstChildTaskID, nil
		}
		return nil, nil
	}

	t, _, _, err := b.locateUnsafe(*id)
	if err != nil {
		return nil, fmt.Errorf("Unable to locate task '%s' (%w)", *id, err)
	}
	if t == nil {
		return nil, fmt.Errorf("Unable to locate task '%s' (but no error!)", *id)
	}

	if len(t.Subtasks) > 0 {
		firstChildTaskID := t.Subtasks[0].ID
		return &firstChildTaskID, nil
	}

	return nil, nil
}
func (b *BacklogYamlIoProvider) GetLastChildTaskID(id *model.TaskID) (*model.TaskID, error) {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	if id == nil {
		if len(b.tasks) > 0 {
			lastChildTaskID := b.tasks[len(b.tasks)-1].ID
			return &lastChildTaskID, nil
		}
		return nil, nil
	}

	t, _, _, err := b.locateUnsafe(*id)
	if err != nil {
		return nil, fmt.Errorf("Unable to locate task '%s' (%w)", *id, err)
	}
	if t == nil {
		return nil, fmt.Errorf("Unable to locate task '%s' (but no error!)", *id)
	}

	if len(t.Subtasks) > 0 {
		lastChildTaskID := t.Subtasks[len(t.Subtasks)-1].ID
		return &lastChildTaskID, nil
	}

	return nil, nil
}

func (b *BacklogYamlIoProvider) GetLocationContext(id model.TaskID) (provider.TaskLocationContext, error) {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	t, _, ctx, err := b.locateUnsafe(id)
	if err != nil {
		return provider.TaskLocationContext{}, fmt.Errorf("Unable to locate task '%s' (%w)", id, err)
	}
	if t == nil {
		return provider.TaskLocationContext{}, fmt.Errorf("Unable to locate task '%s' (but no error!)", id)
	}

	return ctx, nil
}
func (b *BacklogYamlIoProvider) GetCategory(id model.TaskID) (model.CategoryName, error) {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	t, _, _, err := b.locateUnsafe(id)
	if err != nil {
		return "", fmt.Errorf("Unable to locate task '%s' (%w)", id, err)
	}
	if t == nil {
		return "", fmt.Errorf("Unable to locate task '%s' (but no error!)", id)
	}

	return t.Category, nil
}

func (b *BacklogYamlIoProvider) insertRootFrontUnsafe(data model.ReadableTask) (model.TaskID, error) {
	newTask := taskFromReadable(data)
	b.tasks = append([]*model.Task{newTask}, b.tasks...)
	b.dirty = true

	return newTask.ID, nil
}
func (b *BacklogYamlIoProvider) insertRootBackUnsafe(data model.ReadableTask) (model.TaskID, error) {
	newTask := taskFromReadable(data)
	b.tasks = append(b.tasks, newTask)
	b.dirty = true

	return newTask.ID, nil
}
func (b *BacklogYamlIoProvider) InsertBefore(data model.ReadableTask, beforeID model.TaskID) (model.TaskID, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	_, p, _, err := b.locateUnsafe(beforeID)
	if err != nil {
		return "", fmt.Errorf("Unable to locate task '%s' (%w)", beforeID, err)
	}

	newTask := taskFromReadable(data)

	if p == nil {
		idx := slices.IndexFunc(b.tasks, func(rootTask *model.Task) bool { return rootTask.ID == beforeID })
		if idx == -1 {
			return "", fmt.Errorf("Could not find given task in root tasks.")
		}
		b.tasks = slices.Insert(b.tasks, idx, newTask)
		b.dirty = true
		return newTask.ID, nil
	}

	idx := slices.IndexFunc(p.Subtasks, func(childTask *model.Task) bool { return childTask.ID == beforeID })
	if idx == -1 {
		return "", fmt.Errorf("Could not find given task in child tasks of '%s'.", p.ID)
	}
	p.Subtasks = slices.Insert(p.Subtasks, idx, newTask)
	b.dirty = true
	return newTask.ID, nil
}
func (b *BacklogYamlIoProvider) InsertAfter(data model.ReadableTask, afterID model.TaskID) (model.TaskID, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	_, p, _, err := b.locateUnsafe(afterID)
	if err != nil {
		return "", fmt.Errorf("Unable to locate task '%s' (%w)", afterID, err)
	}

	newTask := taskFromReadable(data)

	if p == nil {
		idx := slices.IndexFunc(b.tasks, func(rootTask *model.Task) bool { return rootTask.ID == afterID })
		if idx == -1 {
			return "", fmt.Errorf("Could not find given task in root tasks.")
		}
		b.tasks = slices.Insert(b.tasks, idx+1, newTask)
		b.dirty = true
		return newTask.ID, nil
	}

	idx := slices.IndexFunc(p.Subtasks, func(childTask *model.Task) bool { return childTask.ID == afterID })
	if idx == -1 {
		return "", fmt.Errorf("Could not find given task in child tasks of '%s'.", p.ID)
	}
	p.Subtasks = slices.Insert(p.Subtasks, idx+1, newTask)
	b.dirty = true
	return newTask.ID, nil
}
func (b *BacklogYamlIoProvider) InsertFront(data model.ReadableTask, parentID *model.TaskID) (model.TaskID, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	b.log.Debug().Msgf("in locked area")

	if parentID == nil {
		return b.insertRootFrontUnsafe(data)
	}

	t, _, _, err := b.locateUnsafe(*parentID)
	if err != nil {
		return "", fmt.Errorf("Unable to locate task '%s' (%w)", *parentID, err)
	}

	newTask := taskFromReadable(data)
	t.Subtasks = append([]*model.Task{newTask}, t.Subtasks...)
	b.dirty = true
	return newTask.ID, nil
}
func (b *BacklogYamlIoProvider) InsertBack(data model.ReadableTask, parentID *model.TaskID) (model.TaskID, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if parentID == nil {
		return b.insertRootBackUnsafe(data)
	}

	t, _, _, err := b.locateUnsafe(*parentID)
	if err != nil {
		return "", fmt.Errorf("Unable to locate task '%s' (%w)", *parentID, err)
	}

	newTask := taskFromReadable(data)
	t.Subtasks = append(t.Subtasks, newTask)
	b.dirty = true
	return newTask.ID, nil
}

func (b *BacklogYamlIoProvider) Remove(id model.TaskID) (model.ReadableTask, provider.TaskLocationContext, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	task, parent, ctx, err := b.locateUnsafe(id)
	if err != nil {
		return nil, provider.TaskLocationContext{}, fmt.Errorf("Unable to find task '%s' (%w)", id, err)
	}
	if task == nil {
		return nil, provider.TaskLocationContext{}, fmt.Errorf("No such task.")
	}

	if len(ctx.Parentage) == 0 {
		b.tasks = slices.DeleteFunc(b.tasks, func(t *model.Task) bool { return t.ID == id })
	} else {
		parent.Subtasks = slices.DeleteFunc(parent.Subtasks, func(t *model.Task) bool { return t.ID == id })
	}
	b.dirty = true

	return task, ctx, nil
}

func (b *BacklogYamlIoProvider) Update(id model.TaskID, data model.ReadableTask) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	task, _, _, err := b.locateUnsafe(id)
	if err != nil {
		return fmt.Errorf("Unable to find task '%s' (%w)", id, err)
	}
	if task == nil {
		return fmt.Errorf("No such task.")
	}

	updateTaskFromReader := func(target *model.Task, data model.ReadableTask) error {
		target.Category = data.GetCategory()
		target.Name = data.GetName()
		target.Deadline = data.GetDeadline()
		target.Duration = data.GetDuration()

		subtasks := data.GetSubtasks()
		if len(subtasks) > 0 {
			if len(subtasks) != len(target.Subtasks) {
				return fmt.Errorf("Given data with non-empty subtasks to update task where also subtask number differs from target task; updating subtasks not supported.")
			}
			for i := range subtasks {
				if !target.Subtasks[i].Equal(subtasks[i]) {
					return fmt.Errorf("Non-equal subtasks (by content or sub-subtasks) given in task update data.")
				}
			}
		}
		return nil
	}
	if err := updateTaskFromReader(task, data); err != nil {
		return err
	}
	b.dirty = true

	return nil
}

func (b *BacklogYamlIoProvider) Load() error {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	backlogReader, err := os.Open(b.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.WriteFile(b.filePath, []byte{}, 0644); err != nil {
				return fmt.Errorf("Could not create backlog file at '%s' (%w)", b.filePath, err)
			}
			backlogReader, err = os.Open(b.filePath)
			if err != nil {
				return fmt.Errorf("Unable to create an empty backlog file (%w)", err)
			}
		} else {
			return fmt.Errorf("Unable to read backlog file at '%s' and it is not because of non-existence (%w)", b.filePath, err)
		}
	}
	defer backlogReader.Close()
	if err := b.loadFromReaderUnsafe(backlogReader); err != nil {
		return fmt.Errorf("Unable to load backlog from reader over '%s' (%w)", b.filePath, err)
	}
	b.dirty = false
	return nil
}

func (b *BacklogYamlIoProvider) Save() error {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	writer, err := os.OpenFile(b.filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("unable to write open backlog file '%s' for writing (%w)", b.filePath, err)
	}
	defer writer.Close()

	toBeWritten := BacklogStored{
		TasksByCategory: map[string][]BaseTask{},
	}
	for _, task := range b.tasks {
		categoryName := task.Category
		toBeWritten.TasksByCategory[string(categoryName)] = append(
			toBeWritten.TasksByCategory[string(categoryName)],
			createBaseTaskFromTask(task),
		)
	}

	data, err := yaml.Marshal(toBeWritten)
	if err != nil {
		return fmt.Errorf("unable to marshal backlog (%w)", err)
	}
	_, err = writer.Write(data)
	if err != nil {
		return fmt.Errorf("unable to write to backlog writer (%w)", err)
	}

	b.dirty = false
	return nil
}

func (b *BacklogYamlIoProvider) locateUnsafe(id model.TaskID) (*model.Task, *model.Task, provider.TaskLocationContext, error) {
	for i, rootTask := range b.tasks {
		t, parent, ctx, err := locateInTask(rootTask, id)
		if err != nil {
			return nil, nil, provider.TaskLocationContext{}, fmt.Errorf("Error locating in root task '%s' (%w)", rootTask.ID, err)
		}
		if t != nil {
			if parent == nil {
				if i > 0 {
					prev := b.tasks[i-1].ID
					ctx.Previous = &prev
				}
				if i < (len(b.tasks) - 1) {
					next := b.tasks[i+1].ID
					ctx.Next = &next
				}
				parent = nil // explicitly, there is no parent
			}
			return t, parent, ctx, nil
		}
	}
	return nil, nil, provider.TaskLocationContext{}, nil
}
func locateInTask(t *model.Task, id model.TaskID) (*model.Task, *model.Task, provider.TaskLocationContext, error) {
	if t.ID == id {
		return t, nil, provider.TaskLocationContext{ /* filled by parents */ }, nil
	}
	for i, st := range t.Subtasks {
		if sst, parent, ctx, err := locateInTask(st, id); err != nil {
			return nil, nil, ctx, fmt.Errorf("Error looking in task %s (%w).", st.ID, err)
		} else if sst != nil {
			if parent == nil {
				if i > 0 {
					prev := t.Subtasks[i-1].ID
					ctx.Previous = &prev
				}
				if i < (len(t.Subtasks) - 1) {
					next := t.Subtasks[i+1].ID
					ctx.Next = &next
				}
				parent = t
			}
			ctx.Parentage = append([]model.TaskID{t.ID}, ctx.Parentage...)
			return sst, parent, ctx, nil
		}
	}
	return nil, nil, provider.TaskLocationContext{}, nil
}

func (b *BacklogYamlIoProvider) GetStorageLocationInfo() (string, error) {
	return fmt.Sprintf("yaml-file:%s", b.filePath), nil
}

func (b *BacklogYamlIoProvider) FullyCommitted() (bool, error) {
	b.mtx.RLock()
	defer b.mtx.RUnlock()
	return !b.dirty, nil
}

func (b *BacklogYamlIoProvider) Exists(id model.TaskID) (bool, error) {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	t, _, _, err := b.locateUnsafe(id)
	if err != nil {
		return false, fmt.Errorf("Error finding task (%w)", err)
	}
	return t != nil, nil
}
