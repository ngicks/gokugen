// Code generated by ent, DO NOT EDIT.

package migrate

import (
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/schema/field"
)

var (
	// TasksColumns holds the columns for the "tasks" table.
	TasksColumns = []*schema.Column{
		{Name: "id", Type: field.TypeString, Unique: true},
		{Name: "work_id", Type: field.TypeString},
		{Name: "priority", Type: field.TypeInt, Default: 0},
		{Name: "state", Type: field.TypeEnum, Enums: []string{"scheduled", "dispatched", "cancelled", "done", "err"}, Default: "scheduled"},
		{Name: "err", Type: field.TypeString, Default: ""},
		{Name: "param", Type: field.TypeJSON},
		{Name: "meta", Type: field.TypeJSON},
		{Name: "scheduled_at", Type: field.TypeTime},
		{Name: "created_at", Type: field.TypeTime},
		{Name: "deadline", Type: field.TypeTime, Nullable: true},
		{Name: "cancelled_at", Type: field.TypeTime, Nullable: true},
		{Name: "dispatched_at", Type: field.TypeTime, Nullable: true},
		{Name: "done_at", Type: field.TypeTime, Nullable: true},
	}
	// TasksTable holds the schema information for the "tasks" table.
	TasksTable = &schema.Table{
		Name:       "tasks",
		Columns:    TasksColumns,
		PrimaryKey: []*schema.Column{TasksColumns[0]},
		Indexes: []*schema.Index{
			{
				Name:    "sched",
				Unique:  false,
				Columns: []*schema.Column{TasksColumns[7], TasksColumns[2]},
				Annotation: &entsql.IndexAnnotation{
					DescColumns: map[string]bool{
						TasksColumns[2].Name: true,
					},
				},
			},
			{
				Name:    "task_created_at",
				Unique:  false,
				Columns: []*schema.Column{TasksColumns[8]},
			},
		},
	}
	// Tables holds all the tables in the schema.
	Tables = []*schema.Table{
		TasksTable,
	}
)

func init() {
}
