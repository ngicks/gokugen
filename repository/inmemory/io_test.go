package inmemory

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ngicks/gokugen/def"
	acceptance "github.com/ngicks/gokugen/def/acceptance/repository"
	"github.com/ngicks/und/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIo(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	repo := NewInMemoryRepository()

	for i := 0; i < 100; i++ {
		_, _ = repo.AddTask(context.Background(), def.TaskUpdateParam{
			WorkId:      option.Some("foo"),
			ScheduledAt: option.Some(time.Now()),
		})
	}

	_ = acceptance.CreateEachState(t, repo)

	kv := repo.Save()

	repoOther := NewInMemoryRepository()
	err := repoOther.Load(kv)
	require.NoError(err)

	require.Equal(repo.orderedMap.Len(), repoOther.orderedMap.Len())

	pair := repo.orderedMap.Oldest()
	pairOther := repoOther.orderedMap.Oldest()
	for {
		if pair == nil || pairOther == nil {
			break
		}
		require.Equal(pair.Key, pairOther.Key)
		require.True(pair.Value.Task.Equal(*pairOther.Value.Task))
		pair, pairOther = pair.Next(), pairOther.Next()
	}

	require.Equal(repo.heap.Len(), repoOther.heap.Len())
	for {
		if repo.heap.Len() == 0 || repoOther.heap.Len() == 0 {
			break
		}
		t, u := *repo.heap.Pop().Task, *repoOther.heap.Pop().Task
		assert.True(t.Equal(u), "diff = %s", cmp.Diff(t, u))
	}
}
