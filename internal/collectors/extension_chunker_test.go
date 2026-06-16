package collectors

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hydn-co/mesh-sdk/pkg/catalog/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkExtensionContent_SmallFitsInOneChunk(t *testing.T) {
	chunks := chunkExtensionContent(
		[]entities.AttributeEntry{{Ref: "a", Value: "1"}, {Ref: "b", Value: "2"}},
		[]entities.ClassificationEntry{{Ref: "admin", Confidence: 1}},
		[]entities.RiskFactorEntry{{Ref: "no-mfa", Weight: 5}},
		len("acct-1"),
	)
	require.Len(t, chunks, 1)
	assert.Len(t, chunks[0].Attributes, 2)
	assert.Len(t, chunks[0].Classifications, 1)
	assert.Len(t, chunks[0].RiskFactors, 1)
}

func TestChunkExtensionContent_EmptyYieldsOneEmptyChunk(t *testing.T) {
	chunks := chunkExtensionContent(nil, nil, nil, len("acct-1"))
	require.Len(t, chunks, 1, "every parent yields at least one extension entity")
	assert.Empty(t, chunks[0].Attributes)
}

// A large attribute set must split into multiple chunks, each serializing (as a
// full AccountExtension) under the ceiling, with every item preserved in order.
func TestChunkExtensionContent_LargeSplitsAndStaysUnderLimit(t *testing.T) {
	const n = 400
	bigVal := strings.Repeat("x", 200) // ~200B value each → ~80KB total, forcing splits
	attrs := make([]entities.AttributeEntry, n)
	for i := range attrs {
		attrs[i] = entities.AttributeEntry{Ref: "attr-" + itoa(i), Name: "Attr", Value: bigVal, Type: "string"}
	}

	chunks := chunkExtensionContent(attrs, nil, nil, len("acct-big"))
	require.Greater(t, len(chunks), 1, "expected the oversized set to split into multiple chunks")

	var recovered []entities.AttributeEntry
	for i, chunk := range chunks {
		ext := &entities.AccountExtension{
			AccountRef:  "acct-big",
			Sequence:    i,
			TotalChunks: len(chunks),
			Attributes:  chunk.Attributes,
		}
		data, err := json.Marshal(ext)
		require.NoError(t, err)
		assert.LessOrEqualf(t, len(data), maxExtensionChunkBytes,
			"chunk %d serialized to %d bytes, over the %d ceiling", i, len(data), maxExtensionChunkBytes)
		recovered = append(recovered, chunk.Attributes...)
	}

	require.Len(t, recovered, n, "no attributes lost across chunks")
	for i := range attrs {
		assert.Equal(t, attrs[i].Ref, recovered[i].Ref, "order preserved across chunks")
	}
}

// A single item larger than the ceiling cannot be split further; it is emitted
// alone rather than dropped.
func TestChunkExtensionContent_OversizedSingleItemEmittedAlone(t *testing.T) {
	huge := entities.AttributeEntry{Ref: "huge", Value: strings.Repeat("y", maxExtensionChunkBytes*2)}
	small := entities.AttributeEntry{Ref: "small", Value: "ok"}

	chunks := chunkExtensionContent([]entities.AttributeEntry{huge, small}, nil, nil, 0)
	require.Len(t, chunks, 2, "huge item gets its own chunk, small item the next")
	assert.Equal(t, "huge", chunks[0].Attributes[0].Ref)
	assert.Equal(t, "small", chunks[1].Attributes[0].Ref)
}

// itoa avoids importing strconv just for the fixture loop.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
