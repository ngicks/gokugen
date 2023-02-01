package rescheduler

import (
	"encoding/json"
	"testing"
	"time"
)

func FuzzCronScheduleParam(f *testing.F) {
	f.Add(time.Now().UnixMilli(), time.Now().Add(time.Minute).UnixMilli())
	f.Fuzz(func(t *testing.T, input1, input2 int64) {
		p := CronScheduleParam{
			Prev: time.UnixMilli(input1),
			Next: time.UnixMilli(input2),
		}

		bin, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("MarshalBinary: must not be err. %+v", err)
		}

		var back CronScheduleParam
		err = json.Unmarshal(bin, &back)
		if err != nil {
			t.Fatalf("UnmarshalBinary: must not be err. %+v", err)
		}

		if !(p.Next.Equal(back.Next) && p.Prev.Equal(back.Prev)) {
			t.Fatalf("not equal: left = %+v,\nright = %+v", p, back)
		}
	})
}
func FuzzLimitedScheduleParam(f *testing.F) {
	f.Add(
		time.Now().UnixMilli(),
		[]byte("abcdefghijklmnopqrstuvwxyz1234567890-=[];',./!@#$%^&*()_+{}:|<>?"),
	)
	f.Fuzz(func(t *testing.T, input1 int64, input2 []byte) {
		p := LimitedScheduleParam{
			N:    input1,
			Rest: string(input2),
		}

		bin, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("MarshalBinary: must not be err. %+v", err)
		}

		var back LimitedScheduleParam
		err = json.Unmarshal(bin, &back)
		if err != nil {
			t.Fatalf("UnmarshalBinary: must not be err. %+v", err)
		}

		if !(p.N == back.N && string(p.Rest) == string(back.Rest)) {
			t.Fatalf("not equal: left = %+v,\nright = %+v", p, back)
		}
	})
}
