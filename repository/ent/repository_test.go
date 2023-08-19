package ent

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"
	"net/url"
	"os"
	"path"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/gokugen/def/acceptance/repository"
	"github.com/ngicks/gokugen/repository/ent/gen"
)

var (
	debug = os.Getenv("TEST_DEBUG") == "1"
)

func TestRepository_ent_sqlite(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		panic(err)
	}
	var instances []def.Repository
	defer func() {
		for _, instance := range instances {
			instance.Close()
		}
		// os.RemoveAll(tempDir)
	}()

	newInitializedRepository := func() def.Repository {
		var buf bytes.Buffer
		buf.Grow(16)
		_, err := io.CopyN(&buf, rand.Reader, 16)
		if err != nil {
			panic(err)
		}
		randStr := hex.EncodeToString(buf.Bytes())

		repoFileUrl, _ := url.Parse("file://" + path.Join(tempDir, randStr))
		repoFileUrl.RawQuery = url.Values{"_fk": {"1"}}.Encode()

		t.Logf("creating db file at %s", repoFileUrl.String())

		client, err := gen.Open("sqlite3", repoFileUrl.String())
		if err != nil {
			t.Fatalf("failed opening connection to sqlite: %v", err)
		}

		reposiotory := NewEntRepository(client)

		instances = append(instances, reposiotory)

		// Run the auto migration tool.
		if err := client.Schema.Create(context.Background()); err != nil {
			log.Fatalf("failed creating schema resources: %v", err)
		}
		t.Logf("created %s", repoFileUrl.String())

		return reposiotory
	}

	repository.TestRepository(t, newInitializedRepository, debug)
}
