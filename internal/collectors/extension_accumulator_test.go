package collectors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttrAccumulatorMergesPassesLastWriteWins(t *testing.T) {
	acc := newAttrAccumulator()

	// Pass 1 (grid) then Pass 2 (datastore) for the same parent; the second value
	// for "Department" must win, and a new key is appended.
	acc.add("acc-1", map[string]string{"Department": "Eng", "Zone": "us-east"})
	acc.add("acc-1", map[string]string{"Department": "Platform", "Title": "SWE"})

	entries := acc.attributesFor("acc-1")
	values := map[string]string{}
	for _, e := range entries {
		values[e.Ref] = e.Value
	}
	assert.Equal(t, "Platform", values["Department"], "later pass overwrites the earlier value")
	assert.Equal(t, "us-east", values["Zone"])
	assert.Equal(t, "SWE", values["Title"])
	assert.Len(t, entries, 3, "no duplicate entry for the overwritten key")
}

func TestAttrAccumulatorPreservesFirstSeenOrderAndTouch(t *testing.T) {
	acc := newAttrAccumulator()
	acc.touch("b") // a parent with no attributes still yields an extension
	acc.add("a", map[string]string{"x": "1"})
	acc.add("b", map[string]string{"y": "2"})

	require.Equal(t, []string{"b", "a"}, acc.refs())
	assert.Empty(t, acc.attributesFor("missing"))
}

func TestAttrAccumulatorIgnoresEmptyRef(t *testing.T) {
	acc := newAttrAccumulator()
	acc.touch("")
	acc.add("", map[string]string{"x": "1"})
	assert.Empty(t, acc.refs())
}
