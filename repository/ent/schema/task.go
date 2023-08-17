package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/ngicks/gokugen/def"
)

// Task holds the schema definition for the Task entity.
type Task struct {
	ent.Schema
}

// Fields of the Task.
func (Task) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable(),
		field.String("work_id").
			NotEmpty(),
		field.Int("priority").
			Default(0),
		field.Enum("state").
			Values(def.GetStates()...).
			Default(string(def.TaskScheduled)),
		field.String("err").
			Default(""),
		field.JSON("param", map[string]string{}).
			Default(map[string]string{}),
		field.JSON("meta", map[string]string{}).
			Default(map[string]string{}),
		field.Time("scheduled_at"),
		field.Time("created_at").
			Default(func() time.Time { return def.NormalizeTime(time.Now()) }),
		field.Time("deadline").
			Optional().
			Nillable(),
		field.Time("cancelled_at").
			Optional().
			Nillable(),
		field.Time("dispatched_at").
			Optional().
			Nillable(),
		field.Time("done_at").
			Optional().
			Nillable(),
	}
}

// Edges of the Car.
func (Task) Edges() []ent.Edge {
	return nil
}

func (Task) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("scheduled_at", "priority", "created_at").
			Annotations(entsql.DescColumns("priority")).
			StorageKey("sched"),
		index.Fields("created_at"),
	}
}
