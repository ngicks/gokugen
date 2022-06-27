package repository_test

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/ngicks/gokugen/impl/repository"
	"github.com/ngicks/type-param-common/set"
	"github.com/stretchr/testify/require"
)

func TestRandomStrGen(t *testing.T) {
	// byteLen must be large enough so that no overlapping occurs.
	byteLen := uint(128)
	generator := repository.NewRandStringGenerator(time.Now().UnixMicro(), byteLen, hex.NewEncoder)
	genStrSet := set.Set[string]{}

	for i := 0; i < 1000; i++ {
		s, err := generator.Generate()
		if err != nil {
			t.Fatal(err)
		}
		require.Len(t, s, 128*2)
		genStrSet.Add(s)
	}
	require.Equal(t, genStrSet.Len(), 1000)
}
