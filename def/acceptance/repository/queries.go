package repository

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/und/option"
	"github.com/stretchr/testify/require"
)

func prepareTasks(t *testing.T, repo def.Repository) []def.Task {
	t.Helper()

	require := require.New(t)

	tasks := make([]def.Task, 0)

	eachState := CreateEachState(t, repo)

	tasks = append(
		tasks,
		eachState.Scheduled,  // 0
		eachState.Cancelled,  // 1
		eachState.Dispatched, // 2
		eachState.Done,       // 3
		eachState.Err,        // 4
	)

	for _, param := range []def.TaskUpdateParam{
		{ // 5
			WorkId:      option.Some("baz"),
			Param:       option.Some(map[string]string{"foofoo": "barbar"}),
			Priority:    option.Some(24),
			ScheduledAt: option.Some(sampleDate.Add(3 * time.Hour)),
			Deadline:    option.Some(option.Some(sampleDate.Add(20 * time.Hour))),
			Meta:        option.Some(map[string]string{"bazbaz": "corgecorge"}),
		},
		updateParam.Update(def.TaskUpdateParam{ // 6
			WorkId: option.Some("baz"),
		}),
		updateParam.Update(def.TaskUpdateParam{ // 7
			Param: option.Some(map[string]string{"foofoo": "barbar"}),
		}),
		updateParam.Update(def.TaskUpdateParam{ // 8
			Priority: option.Some(24),
		}),
		updateParam.Update(def.TaskUpdateParam{ // 9
			ScheduledAt: option.Some(sampleDate.Add(3 * time.Hour)),
		}),
		updateParam.Update(def.TaskUpdateParam{ // 10
			Deadline: option.Some(option.Some(sampleDate.Add(20 * time.Hour))),
		}),
		updateParam.Update(def.TaskUpdateParam{ // 11
			Meta: option.Some(map[string]string{"bazbaz": "corgecorge"}),
		}),
		updateParam.Update(def.TaskUpdateParam{ // 12
			Meta: option.Some(map[string]string{
				"foofoo": "barbar",
				"bazbaz": "corgecorge",
				"nah":    "boo",
			}),
			Param: option.Some(map[string]string{
				"foofoo": "barbar",
				"bazbaz": "corgecorge",
				"nah":    "boo",
			}),
		}),
		updateParam.Update(def.TaskUpdateParam{ // 13
			Meta: option.Some(map[string]string{
				"foofoo": "bar",
				"bazbaz": "corge",
			}),
			Param: option.Some(map[string]string{
				"foofoo": "bar",
				"bazbaz": "corge",
			}),
		}),
		updateParam.Update(def.TaskUpdateParam{ // 14
			Meta: option.Some(map[string]string{
				"foofoo": "not_bar",
			}),
			Param: option.Some(map[string]string{
				"foofoo": "not_bar",
			}),
		}),
		updateParam.Update(def.TaskUpdateParam{ // 15
			Deadline: option.Some(option.None[time.Time]()),
		}),
		updateParam.Update(def.TaskUpdateParam{ // 16
			ScheduledAt: option.Some(sampleDate.Add(10 * time.Hour)),
		}),
		updateParam.Update(def.TaskUpdateParam{ // 17
			ScheduledAt: option.Some(sampleDate.Add(15 * time.Hour)),
		}),
		updateParam.Update(def.TaskUpdateParam{ // 18
			ScheduledAt: option.Some(sampleDate.Add(-time.Hour)),
		}),
		updateParam.Update(def.TaskUpdateParam{ // 19
			ScheduledAt: option.Some(sampleDate.Add(-2 * time.Hour)),
		}),
		updateParam.Update(def.TaskUpdateParam{ // 20
			Deadline: option.Some(option.Some(sampleDate.Add(-2 * time.Hour))),
		}),
		updateParam.Update(def.TaskUpdateParam{ // 21
			Deadline: option.Some(option.Some(sampleDate.Add(-time.Hour))),
		}),
	} {
		task, err := repo.AddTask(context.Background(), param)
		require.NoError(err)
		tasks = append(tasks, task)
	}

	return tasks
}

// ensureChange ensures tasks are changed or unchanged as
// `changed` is int slice whose contents are indices of changed tasks of `tasks`.
// As a side effect it re-fetch all tasks and returns them.
func ensureChange(t *testing.T, repo def.Repository, tasks []def.Task, changed []int) []def.Task {
	t.Helper()

	require := require.New(t)

	refetchedTasks := make([]def.Task, len(tasks))

	for idx, task := range tasks {
		refetched, err := repo.GetById(context.Background(), task.Id)
		require.NoError(err)

		refetchedTasks[idx] = refetched

		if slices.Contains(changed, idx) {
			require.False(task.Equal(refetched))
		} else {
			require.True(task.Equal(refetched))
		}
	}

	return refetchedTasks
}

type queryTestSet struct {
	param      def.TaskQueryParam
	shouldFind []int
}

// must be in sync with prepareTasks
func queryTestCases(tasks []def.Task) []queryTestSet {
	var cases []queryTestSet
	cases = append(cases, queryTestCases_state(tasks)...)  // 0 - 5
	cases = append(cases, queryTestCases_simple(tasks)...) // 6 - 11
	cases = append(cases, queryTestCases_map(tasks)...)
	cases = append(cases, queryTestCases_time(tasks)...)
	return cases
}

func queryTestCases_state(tasks []def.Task) []queryTestSet {
	return []queryTestSet{
		// simple cases.
		{ // by id
			param: def.TaskQueryParam{
				Id: option.Some(tasks[2].Id),
			},
			shouldFind: []int{2},
		},
		{ // by combination of id and state
			param: def.TaskQueryParam{
				Id:    option.Some(tasks[0].Id),
				State: option.Some(def.TaskScheduled),
			},
			shouldFind: []int{0},
		},
		{
			param: def.TaskQueryParam{
				State: option.Some(def.TaskCancelled),
			},
			shouldFind: []int{1},
		},
		{
			param: def.TaskQueryParam{
				State: option.Some(def.TaskDispatched),
			},
			shouldFind: []int{2},
		},
		{
			param: def.TaskQueryParam{
				State: option.Some(def.TaskDone),
			},
			shouldFind: []int{3},
		},
		{
			param: def.TaskQueryParam{
				State: option.Some(def.TaskErr),
			},
			shouldFind: []int{4},
		},
	}
}

func queryTestCases_simple(_ []def.Task) []queryTestSet {
	return []queryTestSet{
		{ // 6
			param: def.TaskQueryParam{
				WorkId: option.Some("baz"),
				Param: option.Some(
					[]def.MapMatcher{
						{
							Key:   "foofoo",
							Value: "barbar",
						},
					},
				),
				Priority: option.Some(24),
				ScheduledAt: option.Some(def.TimeMatcher{
					Value: sampleDate.Add(3 * time.Hour),
				}),
				Deadline: option.Some(option.Some(def.TimeMatcher{
					Value: sampleDate.Add(20 * time.Hour),
				})),
				Meta: option.Some([]def.MapMatcher{{
					Key:   "bazbaz",
					Value: "corgecorge",
				}}),
			},
			shouldFind: []int{5},
		},
		{
			param: def.TaskQueryParam{
				WorkId: option.Some("baz"),
			},
			shouldFind: []int{5, 6},
		},
		{
			param: def.TaskQueryParam{
				Param: option.Some(
					[]def.MapMatcher{
						{
							Key:   "foofoo",
							Value: "barbar",
						},
					},
				),
			},
			shouldFind: []int{5, 7, 12},
		},
		{
			param: def.TaskQueryParam{
				Priority: option.Some(24),
			},
			shouldFind: []int{5, 8},
		},
		{
			param: def.TaskQueryParam{
				ScheduledAt: option.Some(def.TimeMatcher{
					Value: sampleDate.Add(3 * time.Hour),
				}),
			},
			shouldFind: []int{5, 9},
		},
		{ // 11
			param: def.TaskQueryParam{
				Meta: option.Some([]def.MapMatcher{{
					Key:   "bazbaz",
					Value: "corgecorge",
				}}),
			},
			shouldFind: []int{5, 11, 12},
		},
	}
}

func queryTestCases_map(_ []def.Task) []queryTestSet {
	return []queryTestSet{
		{ // 12
			param: def.TaskQueryParam{
				Param: option.Some([]def.MapMatcher{
					{
						Key:       "foofoo",
						MatchType: def.MapMatcherHasKey,
					},
				}),
			},
			shouldFind: []int{5, 7, 12, 13, 14},
		},
		{
			param: def.TaskQueryParam{
				Meta: option.Some([]def.MapMatcher{
					{
						Key:       "foofoo",
						Value:     "bar",
						MatchType: def.MapMatcherExact,
					},
				}),
			},
			shouldFind: []int{13},
		},
		{
			param: def.TaskQueryParam{
				Param: option.Some([]def.MapMatcher{
					{
						Key:       "foofoo",
						Value:     "bar",
						MatchType: def.MapMatcherForward,
					},
				}),
			},
			shouldFind: []int{5, 7, 12, 13},
		},
		{
			param: def.TaskQueryParam{
				Param: option.Some([]def.MapMatcher{
					{
						Key:       "foofoo",
						Value:     "bar",
						MatchType: def.MapMatcherBackward,
					},
				}),
			},
			shouldFind: []int{5, 7, 12, 13, 14},
		},
		{
			param: def.TaskQueryParam{
				Meta: option.Some([]def.MapMatcher{
					{
						Key:       "bazbaz",
						Value:     "gec",
						MatchType: def.MapMatcherMiddle,
					},
				}),
			},
			shouldFind: []int{5, 11, 12},
		},
	}
}

func queryTestCases_time(tasks []def.Task) []queryTestSet {
	return []queryTestSet{
		{
			param: def.TaskQueryParam{
				Deadline: option.Some(option.None[def.TimeMatcher]()),
			},
			shouldFind: []int{15},
		},
		{
			param: def.TaskQueryParam{
				Deadline: option.Some(option.Some(def.TimeMatcher{
					Value:     sampleDate.Add(-time.Hour),
					MatchType: def.TimeMatcherEqual,
				})),
			},
			shouldFind: []int{21},
		},
		{
			param: def.TaskQueryParam{
				Deadline: option.Some(option.Some(def.TimeMatcher{
					Value:     sampleDate.Add(-time.Hour),
					MatchType: def.TimeMatcherBefore,
				})),
			},
			shouldFind: []int{20},
		},
		{
			param: def.TaskQueryParam{
				Deadline: option.Some(option.Some(def.TimeMatcher{
					MatchType: def.TimeMatcherNonNull,
				})),
			},
			shouldFind: getExcept(tasks, 15),
		},
		{
			param: def.TaskQueryParam{
				ScheduledAt: option.Some(def.TimeMatcher{
					Value:     sampleDate.Add(-time.Hour),
					MatchType: def.TimeMatcherEqual,
				}),
			},
			shouldFind: []int{18},
		},
		{
			param: def.TaskQueryParam{
				ScheduledAt: option.Some(def.TimeMatcher{
					Value:     sampleDate.Add(-time.Hour),
					MatchType: def.TimeMatcherBefore,
				}),
			},
			shouldFind: []int{19},
		},
		{
			param: def.TaskQueryParam{
				ScheduledAt: option.Some(def.TimeMatcher{
					Value:     sampleDate.Add(-time.Hour),
					MatchType: def.TimeMatcherBeforeEqual,
				}),
			},
			shouldFind: []int{18, 19},
		},
		{
			param: def.TaskQueryParam{
				ScheduledAt: option.Some(def.TimeMatcher{
					Value:     sampleDate.Add(10 * time.Hour),
					MatchType: def.TimeMatcherAfter,
				}),
			},
			shouldFind: []int{17},
		},
		{
			param: def.TaskQueryParam{
				ScheduledAt: option.Some(def.TimeMatcher{
					Value:     sampleDate.Add(10 * time.Hour),
					MatchType: def.TimeMatcherAfterEqual,
				}),
			},
			shouldFind: []int{16, 17},
		},
	}
}

// returns range except excepts indices.
func getExcept(tasks []def.Task, excepts ...int) []int {
	rangeSlice := make([]int, 0, len(tasks)-len(excepts))
	for idx := range tasks {
		if slices.Contains(excepts, idx) {
			continue
		}
		rangeSlice = append(rangeSlice, idx)
	}
	return rangeSlice
}
