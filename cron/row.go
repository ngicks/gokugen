package cron

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/und/option"
	"github.com/robfig/cron/v3"
)

// RawExpression is any of cron standard expression both without and with sec,
// e.g. "15,45 30 12 *" or "@every 4h25m".
// Or JsonExp.
type RawExpression any

func ParseRawExpression(raw RawExpression) (cron.Schedule, error) {
	switch x := raw.(type) {
	case JsonExp:
		return x.Parse()
	case string:
		if len(x) > 0 && !strings.Contains(x, "@") {
			scanner := bufio.NewScanner(strings.NewReader(x))
			scanner.Split(bufio.ScanWords)
			var count int
			for scanner.Scan() {
				text := scanner.Text()
				if !strings.HasPrefix(text, "TZ=") &&
					!strings.HasPrefix(text, "CRON_TZ=") {
					count++
				}
			}
			if count == 6 {
				return parser.Parse(x)
			}
		}
		return cron.ParseStandard(x)
	}

	return nil, fmt.Errorf(
		"ParseRawExpression: unknown. input must be string or JsonExp, but is %T",
		raw,
	)
}

type JsonExp struct {
	Second, Minute, Hour, Dom, Month, Dow []uint64

	// Override location for this schedule.
	Location string
}

func (e JsonExp) Format() string {
	var buf bytes.Buffer

	if e.Location != "" {
		buf.WriteString("TZ=")
		buf.WriteString(e.Location)
		buf.WriteByte(' ')
	}

	for _, nums := range [...][]uint64{
		e.Second, e.Minute, e.Hour, e.Dom, e.Month, e.Dow,
	} {
		if len(nums) == 0 {
			buf.WriteByte('*')
		} else {
			for _, num := range nums {
				buf.WriteString(strconv.FormatUint(num, 10))
				buf.WriteByte(',')
			}
			buf.Truncate(buf.Len() - 1)
		}
		buf.WriteByte(' ')
	}
	buf.Truncate(buf.Len() - 1)
	return buf.String()
}

var (
	parser = cron.NewParser(
		cron.Second |
			cron.Minute |
			cron.Hour |
			cron.Dom |
			cron.Month |
			cron.Dow,
	)
)

func (e JsonExp) Parse() (cron.Schedule, error) {
	return parser.Parse(e.Format())
}

type RowRaw struct {
	Param    def.TaskUpdateParam
	Schedule RawExpression
}

func (r RowRaw) Parse() (Row, error) {
	sched, err := ParseRawExpression(r.Schedule)

	if err != nil {
		return Row{}, err
	}
	return Row{
		Param:    r.Param.Clone(),
		Schedule: sched,
	}, nil
}

type Row struct {
	Param    def.TaskUpdateParam
	Schedule cron.Schedule
}

func (r Row) Next(prev time.Time) def.TaskUpdateParam {
	return r.Param.Update(def.TaskUpdateParam{ScheduledAt: option.Some(r.Schedule.Next(prev))})
}
