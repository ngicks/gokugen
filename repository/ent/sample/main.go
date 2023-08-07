package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/ngicks/gokugen/repository/ent/gen"
)

func main() {
	client, err := gen.Open("sqlite3", "./test.db?_fk=1")
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}
	defer client.Close()
	// Run the auto migration tool.
	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	fmt.Println(
		client.Task.Create().
			SetID(uuid.NewString()).
			SetWorkID("foo").
			SetScheduledAt(time.Now()).
			Exec(context.Background()),
	)
}
