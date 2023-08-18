// Code generated by ent, DO NOT EDIT.

package gen

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/ngicks/gokugen/repository/ent/gen/task"
)

// Task is the model entity for the Task schema.
type Task struct {
	config `json:"-"`
	// ID of the ent.
	ID string `json:"id,omitempty"`
	// WorkID holds the value of the "work_id" field.
	WorkID string `json:"work_id,omitempty"`
	// Priority holds the value of the "priority" field.
	Priority int `json:"priority,omitempty"`
	// State holds the value of the "state" field.
	State task.State `json:"state,omitempty"`
	// Err holds the value of the "err" field.
	Err string `json:"err,omitempty"`
	// Param holds the value of the "param" field.
	Param map[string]string `json:"param,omitempty"`
	// Meta holds the value of the "meta" field.
	Meta map[string]string `json:"meta,omitempty"`
	// ScheduledAt holds the value of the "scheduled_at" field.
	ScheduledAt time.Time `json:"scheduled_at,omitempty"`
	// CreatedAt holds the value of the "created_at" field.
	CreatedAt time.Time `json:"created_at,omitempty"`
	// Deadline holds the value of the "deadline" field.
	Deadline *time.Time `json:"deadline,omitempty"`
	// CancelledAt holds the value of the "cancelled_at" field.
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`
	// DispatchedAt holds the value of the "dispatched_at" field.
	DispatchedAt *time.Time `json:"dispatched_at,omitempty"`
	// DoneAt holds the value of the "done_at" field.
	DoneAt       *time.Time `json:"done_at,omitempty"`
	selectValues sql.SelectValues
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Task) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case task.FieldParam, task.FieldMeta:
			values[i] = new([]byte)
		case task.FieldPriority:
			values[i] = new(sql.NullInt64)
		case task.FieldID, task.FieldWorkID, task.FieldState, task.FieldErr:
			values[i] = new(sql.NullString)
		case task.FieldScheduledAt, task.FieldCreatedAt, task.FieldDeadline, task.FieldCancelledAt, task.FieldDispatchedAt, task.FieldDoneAt:
			values[i] = new(sql.NullTime)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Task fields.
func (t *Task) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case task.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				t.ID = value.String
			}
		case task.FieldWorkID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field work_id", values[i])
			} else if value.Valid {
				t.WorkID = value.String
			}
		case task.FieldPriority:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field priority", values[i])
			} else if value.Valid {
				t.Priority = int(value.Int64)
			}
		case task.FieldState:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field state", values[i])
			} else if value.Valid {
				t.State = task.State(value.String)
			}
		case task.FieldErr:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field err", values[i])
			} else if value.Valid {
				t.Err = value.String
			}
		case task.FieldParam:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field param", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &t.Param); err != nil {
					return fmt.Errorf("unmarshal field param: %w", err)
				}
			}
		case task.FieldMeta:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field meta", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &t.Meta); err != nil {
					return fmt.Errorf("unmarshal field meta: %w", err)
				}
			}
		case task.FieldScheduledAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field scheduled_at", values[i])
			} else if value.Valid {
				t.ScheduledAt = value.Time
			}
		case task.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				t.CreatedAt = value.Time
			}
		case task.FieldDeadline:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deadline", values[i])
			} else if value.Valid {
				t.Deadline = new(time.Time)
				*t.Deadline = value.Time
			}
		case task.FieldCancelledAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field cancelled_at", values[i])
			} else if value.Valid {
				t.CancelledAt = new(time.Time)
				*t.CancelledAt = value.Time
			}
		case task.FieldDispatchedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field dispatched_at", values[i])
			} else if value.Valid {
				t.DispatchedAt = new(time.Time)
				*t.DispatchedAt = value.Time
			}
		case task.FieldDoneAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field done_at", values[i])
			} else if value.Valid {
				t.DoneAt = new(time.Time)
				*t.DoneAt = value.Time
			}
		default:
			t.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Task.
// This includes values selected through modifiers, order, etc.
func (t *Task) Value(name string) (ent.Value, error) {
	return t.selectValues.Get(name)
}

// Update returns a builder for updating this Task.
// Note that you need to call Task.Unwrap() before calling this method if this Task
// was returned from a transaction, and the transaction was committed or rolled back.
func (t *Task) Update() *TaskUpdateOne {
	return NewTaskClient(t.config).UpdateOne(t)
}

// Unwrap unwraps the Task entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (t *Task) Unwrap() *Task {
	_tx, ok := t.config.driver.(*txDriver)
	if !ok {
		panic("gen: Task is not a transactional entity")
	}
	t.config.driver = _tx.drv
	return t
}

// String implements the fmt.Stringer.
func (t *Task) String() string {
	var builder strings.Builder
	builder.WriteString("Task(")
	builder.WriteString(fmt.Sprintf("id=%v, ", t.ID))
	builder.WriteString("work_id=")
	builder.WriteString(t.WorkID)
	builder.WriteString(", ")
	builder.WriteString("priority=")
	builder.WriteString(fmt.Sprintf("%v", t.Priority))
	builder.WriteString(", ")
	builder.WriteString("state=")
	builder.WriteString(fmt.Sprintf("%v", t.State))
	builder.WriteString(", ")
	builder.WriteString("err=")
	builder.WriteString(t.Err)
	builder.WriteString(", ")
	builder.WriteString("param=")
	builder.WriteString(fmt.Sprintf("%v", t.Param))
	builder.WriteString(", ")
	builder.WriteString("meta=")
	builder.WriteString(fmt.Sprintf("%v", t.Meta))
	builder.WriteString(", ")
	builder.WriteString("scheduled_at=")
	builder.WriteString(t.ScheduledAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(t.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := t.Deadline; v != nil {
		builder.WriteString("deadline=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	if v := t.CancelledAt; v != nil {
		builder.WriteString("cancelled_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	if v := t.DispatchedAt; v != nil {
		builder.WriteString("dispatched_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	if v := t.DoneAt; v != nil {
		builder.WriteString("done_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteByte(')')
	return builder.String()
}

// Tasks is a parsable slice of Task.
type Tasks []*Task