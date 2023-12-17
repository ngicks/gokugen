package cron

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/und/option"
)

func TestEntry_schedule_hash(t *testing.T) {
	type testCase struct {
		schedule Schedule
		meta     map[string]string
	}

	assertNoDiff := func(param def.TaskUpdateParam, tc testCase) {
		t.Helper()
		if diff := cmp.Diff(param.Meta.Value(), tc.meta); diff != "" {
			t.Fatalf("not equal. diff = %s", diff)
		}
	}

	for _, tc := range []testCase{
		{
			schedule: must(RowRaw{
				Param: def.TaskUpdateParam{
					WorkId: option.Some("foo"),
				},
				Schedule: "@every 1h",
			}.Parse()),
			meta: map[string]string{
				MetaKeyScheduleHash: "@every 1h",
			},
		},
		{
			schedule: must(RowRaw{
				Param: def.TaskUpdateParam{
					WorkId: option.Some("bar"),
					Meta: option.Some(map[string]string{
						"foo": "bar",
					}),
				},
				Schedule: JsonExp{
					Second:   []uint64{0},
					Minute:   []uint64{0},
					Hour:     []uint64{16},
					Location: "Asia/Tokyo",
				},
			}.Parse()),
			meta: map[string]string{
				"foo":               "bar",
				MetaKeyScheduleHash: "TZ=Asia/Tokyo 0 0 16 * * *",
			},
		},
	} {
		ent := NewEntry(fakeCurrent, tc.schedule)
		assertNoDiff(ent.Param(), tc)
		assertNoDiff(ent.Next(), tc)
		assertNoDiff(ent.Next(), tc)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
