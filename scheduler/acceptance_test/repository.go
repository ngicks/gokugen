package acceptancetest

import (
	"testing"

	"github.com/ngicks/gokugen/scheduler"
)

type RepositoryTestConfig struct {
	FindMetaContain  FindMetaContainTestConfig
	DeleteBefore     bool
	RevertDispatched bool // Setting RevertDispatched true let TestRepository test that optional method.
}

// TestRepository is an exported acceptance test which calls all relevant tests defined within this package.
// An implementation would only be considered as a conformant if it passed this test.
// Each individual tests, which are functions prefixed with TestRepository_, are exported only for debugging purpose.
//
// Implementations may call this test with their own implementation in their own test file.
// Some, not all, tests needs a fresh repository instance for each test case.
// For those, repoFactory must return a newly created empty repository.
//
// Some of interface's feature is optional. Callers can utilize cfg to enable tests for those optional functionalities.
func TestRepository(t *testing.T, repoFactory func() scheduler.TaskRepository, cfg RepositoryTestConfig) {
	repo := repoFactory()
	now := TruncatedNow()

	t.Run("GetNext on empty Repository returns Empty Repository Error", func(t *testing.T) {
		TestRepository_GetNext_on_empty_repo(t, repo)
	})

	t.Run("AddTask", func(t *testing.T) {
		TestRepository_AddTask(t, repo, now)
	})

	t.Run("GetById", func(t *testing.T) {
		TestRepository_GetById(t, repo, now)
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("update only non-zero param fields", func(t *testing.T) {
			TestRepository_Update_only_non_zero_param_fields(t, repo, now)
		})

		t.Run("trying to update id that is already unable-to-update state", func(t *testing.T) {
			TestRepository_Update_error_on_non_updatable(t, repo, now)
		})

		t.Run("If param only has Meta, update is allowed", func(t *testing.T) {
			TestRepository_Update_param_only_is_always_allowed(t, repo, now)
		})

		t.Run("trying to update nonexistent id", func(t *testing.T) {
			TestRepository_Update_error_on_nonexistent(t, repo, now)
		})
	})

	t.Run("Cancel", func(t *testing.T) {
		t.Run("cancel", func(t *testing.T) {
			TestRepository_Cancel(t, repo, now)
		})

		t.Run("trying to cancel id that is already unable-to-update state", func(t *testing.T) {
			TestRepository_Cancel_error_on_non_updatable(t, repo, now)
		})

		t.Run("trying to cancel nonexistent id", func(t *testing.T) {
			TestRepository_Cancel_error_on_nonexistent(t, repo)
		})
	})

	t.Run("MarkAsDispatched", func(t *testing.T) {
		t.Run(
			"trying to mark-as-dispatched id that is already unable-to-update state",
			func(t *testing.T) {
				TestRepository_MarkAsDispatched_error_on_non_updatable(t, repo, now)
			},
		)

		t.Run("trying to mark-as-dispatched nonexistent id", func(t *testing.T) {
			TestRepository_MarkAsDispatched_error_on_nonexistent(t, repo, now)
		})
	})

	t.Run("MarkAsDone", func(t *testing.T) {
		t.Run(
			"only eligible state for MarkAsDone is marked-as-dispatched state",
			func(t *testing.T) {
				TestRepository_MarkAsDone_error_on_not_dispatched(t, repo, now)
			},
		)

		t.Run("MarkAsDone with non-nil error will set error string", func(t *testing.T) {
			TestRepository_MarkAsDone_with_error(t, repo, now)
		})

		t.Run(
			"trying to mark-as-done id that is already unable-to-update state",
			func(t *testing.T) {
				TestRepository_MarkAsDone_error_on_non_updatable(t, repo, now)
			},
		)

		t.Run("trying to mark-as-done nonexistent id", func(t *testing.T) {
			TestRepository_MarkAsDone_error_on_nonexistent(t, repo, now)
		})
	})

	t.Run("Find", func(t *testing.T) {
		TestRepository_Find(t, repo)
	})

	t.Run("FindMetaContain", func(t *testing.T) {
		TestRepository_FindMetaContain(
			t,
			repo,
			cfg.FindMetaContain,
		)
	})

	if cfg.RevertDispatched {
		t.Run("RevertDispatched", func(t *testing.T) {
			TestRepository_RevertDispatched(t, repo)
		})
	}

	if cfg.DeleteBefore {
		t.Run("DeleteBefore", func(t *testing.T) {
			TestRepository_DeleteBefore(t, repoFactory())
		})
	}

	t.Run("normal usecase, sequence of AddTask, Pop, MarkAsDispatched,"+
		" and MarkAsFailed or MarkAsDone",
		func(t *testing.T) {
			TestRepository_normal_usecase(t, repo)
		},
	)

	t.Run("timer is updated if min element is affected", func(t *testing.T) {
		t.Run("AddTask", func(t *testing.T) {
			TestRepository_timer_update_AddTask(t, repo)
		})

		t.Run("Update", func(t *testing.T) {
			TestRepository_timer_update_Update(t, repo)
		})

		t.Run("Cancel", func(t *testing.T) {
			TestRepository_timer_update_Cancel(t, repo)
		})

		t.Run("MarkAsDispatched", func(t *testing.T) {
			TestRepository_timer_update_MarkAsDispatched(t, repo)
		})
	})

	t.Run("Start and Stop timer", func(t *testing.T) {
		TestRepository_timer(t, repo)
	})
}
