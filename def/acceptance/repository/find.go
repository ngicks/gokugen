package repository

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ngicks/gokugen/def"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_can_find_tasks(t *testing.T, repo def.Repository) {
	t.Helper()
	assert := assert.New(t)
	require := require.New(t)

	tasks := prepareTasks(t, repo)

	var (
		found []def.Task
		err   error
	)
	found, err = repo.Find(context.Background(), def.TaskQueryParam{}, 0, -1)
	require.NoError(err)
	require.Len(found, len(tasks))

	for idx, set := range queryTestCases(tasks) {
		t.Logf("case = %d, query = %s",
			idx, mustMarshal(set.param),
		)
		found, err = repo.Find(
			context.Background(),
			set.param,
			0, -1,
		)
		require.NoError(err)
		if !assert.Len(found, len(set.shouldFind)) {
			t.Logf("found = %s", mustMarshal(found))
		}
		for order, index := range set.shouldFind {
			assert.True(
				tasks[index].Equal(found[order]),
				"diff = %s", cmp.Diff(tasks[index], found[order]),
			)
		}
	}

	_ = ensureChange(t, repo, tasks, nil)
}

func mustMarshal(v any) []byte {
	bin, _ := json.Marshal(v)
	return bin
}
