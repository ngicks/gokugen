package cron_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/ngicks/gokugen/cron"
)

func TestCron(t *testing.T) {
	t.Run("building cron row", func(t *testing.T) {
		// Sunday
		pseudoNow := time.Date(2022, time.April, 10, 0, 0, 0, 0, time.UTC)
		builder := cron.NewBuilder().Command([]string{"ls"})

		t.Run("weekly", func(t *testing.T) {
			sched, _ := builder.Weekly(time.Monday, 11, 23).Build()
			targetDate, _ := sched.NextSchedule(pseudoNow)

			if expected := pseudoNow.AddDate(0, 0, 1).Add(11*time.Hour + 23*time.Minute); expected != targetDate {
				t.Fatalf("wronge date! expected=%s:%s, actual=%s:%s", expected, expected.Weekday(), targetDate, targetDate.Weekday())
			}
		})

		t.Run("yearly", func(t *testing.T) {
			sched, _ := builder.Yearly(time.January, 23, 2, 15).Build()
			targetDate, _ := sched.NextSchedule(pseudoNow)

			if expected := pseudoNow.AddDate(1, -3, 13).Add(2*time.Hour + 15*time.Minute); expected != targetDate {
				t.Fatalf("wronge date! expected=%s, actual=%s", expected, targetDate)
			}
		})

		t.Run("monthly", func(t *testing.T) {
			sched, _ := builder.Monthly(9, 23, 59).Build()
			targetDate, _ := sched.NextSchedule(pseudoNow)

			if expected := pseudoNow.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute); expected != targetDate {
				t.Fatalf("wronge date! expected=%s, actual=%s", expected, targetDate)
			}
		})

		t.Run("daily", func(t *testing.T) {
			sched, _ := builder.Daily(3, 15).Build()
			targetDate, _ := sched.NextSchedule(pseudoNow)

			if expected := pseudoNow.AddDate(0, 0, 0).Add(3*time.Hour + 15*time.Minute); expected != targetDate {
				t.Fatalf("wronge date! expected=%s, actual=%s", expected, targetDate)
			}
		})

		t.Run("hourly", func(t *testing.T) {
			sched, _ := builder.Hourly(23).Build()
			targetDate, _ := sched.NextSchedule(pseudoNow)

			if expected := pseudoNow.AddDate(0, 0, 0).Add(23 * time.Minute); expected != targetDate {
				t.Fatalf("wronge date! expected=%s, actual=%s", expected, targetDate)
			}
		})

		t.Run("reboot", func(t *testing.T) {
			sched, _ := builder.Reboot().Build()
			targetDate, _ := sched.NextSchedule(pseudoNow)

			if targetDate != pseudoNow {
				t.Fatalf("wronge date! expected=%s, actual=%s", pseudoNow, targetDate)
			}
		})
	})

	t.Run("invalid range", func(t *testing.T) {
		b := cron.NewBuilder().Command([]string{"ls"})
		nilWork, _ := cron.NewBuilder().Yearly(time.April, 1, 1, 1).Build()
		testCases := []cron.Row{
			nilWork,
			b.Month(0).MustBuild(),
			b.Month(13).MustBuild(),
			b.Month(187).MustBuild(),
			b.Day(0).MustBuild(),
			b.Day(32).MustBuild(),
			b.Day(67).MustBuild(),
			b.Hour(24).MustBuild(),
			b.Hour(32).MustBuild(),
			b.Minute(60).MustBuild(),
			b.Minute(70).MustBuild(),
		}
		for _, testCase := range testCases {
			if _, err := testCase.NextSchedule(time.Now()); err == nil {
				t.Fatalf("must not nil")
			}
		}
	})

	t.Run("leap year", func(t *testing.T) {
		c := cron.NewBuilder().Command([]string{"ls"}).Yearly(time.February, 31, 0, 0).MustBuild()
		for i := 2022; i < 2022+100; i++ {
			target := time.Date(i, time.January, 1, 0, 0, 0, 0, time.UTC)
			sched, _ := c.NextSchedule(target)
			if isLeapYear(i) {
				if sched.Day() != 29 {
					t.Fatalf("invalid leap year treatment: must be %d, but is %s", 29, sched.Format(time.RFC3339Nano))
				}
			} else {
				if sched.Day() != 28 {
					t.Fatalf("invalid leap year treatment: must be %d, but is %s", 28, sched.Format(time.RFC3339Nano))
				}
			}
		}
	})

	t.Run("monthly: overflow of 30th and 31st", func(t *testing.T) {
		testMonthly := func(target time.Time, row cron.Row) {
			for i := 1; i <= 12*4; i++ {
				added := target.AddDate(0, i, 0)
				fmt.Println("case:", added.Format(time.RFC3339Nano))
				sched, _ := row.NextSchedule(added)

				var expectedDayMax int
				switch sched.Month() {
				// 28 | 29
				case time.February:
					year := sched.Year()
					if isLeapYear(year) {
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
				if sched.Day() != expectedDay {
					t.Fatalf("invalid monthly year treatment: must be %d, but is %s", expectedDay, sched.Format(time.RFC3339Nano))
				}
			}
		}

		target := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
		testMonthly(target, cron.NewBuilder().Command([]string{"ls"}).Monthly(31, 12, 30).MustBuild())
		testMonthly(target, cron.NewBuilder().Command([]string{"ls"}).Monthly(30, 12, 30).MustBuild())
	})

	t.Run("multiple value", func(t *testing.T) {
		b := cron.NewBuilder().Command([]string{"ls"})
		pseudoNow := time.Date(2022, time.April, 10, 0, 0, 0, 0, time.UTC)

		row := b.WeekDay(time.Saturday, time.Tuesday, time.Thursday).Hour(0).Minute(0).MustBuild()

		next := pseudoNow

		next, _ = row.NextSchedule(next)
		if expected := pseudoNow.AddDate(0, 0, 2); expected != next {
			t.Fatalf(
				"invalid monthly year treatment: must be %s, but is %s",
				expected.Format(time.RFC3339Nano),
				next.Format(time.RFC3339Nano),
			)
		}
		next, _ = row.NextSchedule(next)
		if expected := pseudoNow.AddDate(0, 0, 4); expected != next {
			t.Fatalf(
				"invalid monthly year treatment: must be %s, but is %s",
				expected.Format(time.RFC3339Nano),
				next.Format(time.RFC3339Nano),
			)
		}
		next, _ = row.NextSchedule(next)
		if expected := pseudoNow.AddDate(0, 0, 6); expected != next {
			t.Fatalf(
				"invalid monthly year treatment: must be %s, but is %s",
				expected.Format(time.RFC3339Nano),
				next.Format(time.RFC3339Nano),
			)
		}

		row = b.Hour(12, 5, 7).Minute(0).MustBuild()
		next = pseudoNow

		next, _ = row.NextSchedule(next)
		if expected := pseudoNow.Add(5 * time.Hour); expected != next {
			t.Fatalf(
				"invalid monthly year treatment: must be %s, but is %s",
				expected.Format(time.RFC3339Nano),
				next.Format(time.RFC3339Nano),
			)
		}
		next, _ = row.NextSchedule(next)
		if expected := pseudoNow.Add(7 * time.Hour); expected != next {
			t.Fatalf(
				"invalid monthly year treatment: must be %s, but is %s",
				expected.Format(time.RFC3339Nano),
				next.Format(time.RFC3339Nano),
			)
		}
		next, _ = row.NextSchedule(next)
		if expected := pseudoNow.Add(12 * time.Hour); expected != next {
			t.Fatalf(
				"invalid monthly year treatment: must be %s, but is %s",
				expected.Format(time.RFC3339Nano),
				next.Format(time.RFC3339Nano),
			)
		}
	})
}

func isLeapYear(year int) bool {
	return year%400 == 0 || (year%100 != 0 && year%4 == 0)
}
