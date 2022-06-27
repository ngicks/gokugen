package cron_test

import (
	"testing"
	"time"

	"github.com/ngicks/gokugen/cron"
	"github.com/stretchr/testify/require"
)

func TestCron(t *testing.T) {
	t.Parallel()
	t.Run("building cron row", func(t *testing.T) {
		t.Parallel()
		// Sunday
		pseudoNow := time.Date(2022, time.April, 10, 0, 0, 0, 0, time.UTC)
		builder := cron.Builder{}.Command([]string{"ls"})

		t.Run("weekly", func(t *testing.T) {
			sched := builder.Weekly([]time.Weekday{time.Monday}, []uint8{11}, []uint8{23}).Build()
			targetDate, _ := sched.NextSchedule(pseudoNow)

			expected := pseudoNow.AddDate(0, 0, 1).Add(11*time.Hour + 23*time.Minute)
			require.Equal(t, expected, targetDate)
		})

		t.Run("yearly", func(t *testing.T) {
			sched := builder.Yearly([]time.Month{time.January}, []uint8{23}, []uint8{2}, []uint8{15}).Build()
			targetDate, _ := sched.NextSchedule(pseudoNow)

			expected := pseudoNow.AddDate(1, -3, 13).Add(2*time.Hour + 15*time.Minute)
			require.Equal(t, expected, targetDate)
		})

		t.Run("monthly", func(t *testing.T) {
			sched := builder.Monthly([]uint8{9}, []uint8{23}, []uint8{59}).Build()
			targetDate, _ := sched.NextSchedule(pseudoNow)

			expected := pseudoNow.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute)
			require.Equal(t, expected, targetDate)
		})

		t.Run("daily", func(t *testing.T) {
			sched := builder.Daily([]uint8{3}, []uint8{15}).Build()
			targetDate, _ := sched.NextSchedule(pseudoNow)

			expected := pseudoNow.AddDate(0, 0, 0).Add(3*time.Hour + 15*time.Minute)
			require.Equal(t, expected, targetDate)
		})

		t.Run("hourly", func(t *testing.T) {
			sched := builder.Hourly([]uint8{23}).Build()
			targetDate, _ := sched.NextSchedule(pseudoNow)

			expected := pseudoNow.AddDate(0, 0, 0).Add(23 * time.Minute)
			require.Equal(t, expected, targetDate)
		})

		t.Run("reboot", func(t *testing.T) {
			sched := builder.Reboot().Build()
			targetDate, _ := sched.NextSchedule(pseudoNow)

			require.Equal(t, pseudoNow, targetDate)
		})
	})

	t.Run("invalid range", func(t *testing.T) {
		b := cron.Builder{}.Command([]string{"ls"})
		nilWork := cron.Builder{}.Yearly([]time.Month{time.April}, []uint8{1}, []uint8{1}, []uint8{1}).Build()
		testCases := []cron.Row{
			nilWork,
			b.Month(0).Build(),
			b.Month(13).Build(),
			b.Month(187).Build(),
			b.Day(0).Build(),
			b.Day(32).Build(),
			b.Day(67).Build(),
			b.Hour(24).Build(),
			b.Hour(32).Build(),
			b.Minute(60).Build(),
			b.Minute(70).Build(),
		}
		for _, testCase := range testCases {
			_, err := testCase.NextSchedule(time.Now())
			require.Error(t, err)
		}
	})

	t.Run("leap year", func(t *testing.T) {
		c := cron.Builder{}.
			Command([]string{"ls"}).
			Yearly([]time.Month{time.February}, []uint8{31}, []uint8{0}, []uint8{0}).
			Build()
		for i := 2022; i < 2022+100; i++ {
			target := time.Date(i, time.January, 1, 0, 0, 0, 0, time.UTC)
			sched, _ := c.NextSchedule(target)
			if cron.IsLeapYear(time.Date(i, 1, 1, 0, 0, 0, 0, time.UTC)) {
				require.Equal(t, sched.Day(), 29)
			} else {
				require.Equal(t, sched.Day(), 28)
			}
		}
	})

	t.Run("monthly: overflow of 30th and 31st", func(t *testing.T) {
		testMonthly := func(target time.Time, row cron.Row) {
			for i := 1; i <= 12*4; i++ {
				added := target.AddDate(0, i, 0)
				// fmt.Println("case:", added.Format(time.RFC3339Nano))
				sched, _ := row.NextSchedule(added)

				var expectedDayMax int
				switch sched.Month() {
				// 28 | 29
				case time.February:
					if cron.IsLeapYear(sched) {
						expectedDayMax = 29
					} else {
						expectedDayMax = 28
					}
				// 30
				case time.April, time.June, time.September, time.November:
					expectedDayMax = 30
				// 31
				case time.January, time.March, time.May, time.July, time.August, time.October, time.December:
					expectedDayMax = 31
				default:
					t.Fatalf("unknown month: %d", sched.Month())
				}

				var expectedDay int
				if row.Day != nil && uint((*row.Day)[0]) < uint(expectedDayMax) {
					expectedDay = int((*row.Day)[0])
				} else {
					expectedDay = expectedDayMax
				}

				require.Equal(t, expectedDay, sched.Day())
			}
		}

		target := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
		testMonthly(
			target,
			cron.Builder{}.Command([]string{"ls"}).Monthly([]uint8{31}, []uint8{12}, []uint8{30}).Build(),
		)
		testMonthly(
			target,
			cron.Builder{}.Command([]string{"ls"}).Monthly([]uint8{30}, []uint8{12}, []uint8{30}).Build(),
		)
	})

	t.Run("multiple value", func(t *testing.T) {
		b := cron.Builder{}.Command([]string{"ls"})
		pseudoNow := time.Date(2022, time.April, 10, 0, 0, 0, 0, time.UTC)

		row := b.WeekDay(time.Saturday, time.Tuesday, time.Thursday).Hour(0).Minute(0).Build()

		next := pseudoNow

		next, _ = row.NextSchedule(next)
		require.Equal(t, pseudoNow.AddDate(0, 0, 2), next)
		next, _ = row.NextSchedule(next)
		require.Equal(t, pseudoNow.AddDate(0, 0, 4), next)
		next, _ = row.NextSchedule(next)
		require.Equal(t, pseudoNow.AddDate(0, 0, 6), next)

		row = b.Hour(12, 5, 7).Minute(0).Build()
		next = pseudoNow

		next, _ = row.NextSchedule(next)
		require.Equal(t, pseudoNow.Add(5*time.Hour), next)
		next, _ = row.NextSchedule(next)
		require.Equal(t, pseudoNow.Add(7*time.Hour), next)
		next, _ = row.NextSchedule(next)
		require.Equal(t, pseudoNow.Add(12*time.Hour), next)
	})
}
