package commands

import (
	"context"
	"fmt"

	"gtask/internal/service"
)

type taskPageCache map[string]map[int][]service.Task // listID -> page -> tasks

// findTaskByNumberCached finds a task by its 1-based number in a list, caching
// pages (100 tasks per page) to avoid redundant backend calls.
func findTaskByNumberCached(ctx context.Context, svc service.Service, listID string, num int, cache taskPageCache) (service.Task, error) {
	const pageSize = 100

	page := (num-1)/pageSize + 1
	indexInPage := (num - 1) % pageSize

	if cache == nil {
		cache = make(taskPageCache)
	}
	if cache[listID] == nil {
		cache[listID] = make(map[int][]service.Task)
	}

	tasks, ok := cache[listID][page]
	if !ok {
		var err error
		tasks, err = svc.ListOpenTasks(ctx, listID, page)
		if err != nil {
			return service.Task{}, err
		}
		cache[listID][page] = tasks
	}

	if indexInPage >= len(tasks) {
		return service.Task{}, fmt.Errorf("task number out of range: %d", num)
	}

	return tasks[indexInPage], nil
}

