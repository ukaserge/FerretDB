// Copyright 2021 FerretDB Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/FerretDB/FerretDB/integration/shareddata"
)

func TestQueryProjection(t *testing.T) {
	t.Parallel()
	providers := []shareddata.Provider{shareddata.Composites}
	ctx, collection := setup(t, providers...)

	_, err := collection.InsertMany(ctx, []any{
		bson.D{
			{"_id", "document-composite-2"},
			{"value", bson.A{
				bson.D{{"field", int32(42)}},
				bson.D{{"field", int32(44)}},
			}},
		},
	})
	require.NoError(t, err)

	for name, tc := range map[string]struct {
		projection any
		filter     any
		expected   bson.D
	}{
		"FindProjectionInclusions": {
			filter: bson.D{{"_id", "document-composite"}},
			// TODO: https://github.com/FerretDB/FerretDB/issues/537
			projection: bson.D{{"foo", int32(1)}, {"42", true}},
			expected:   bson.D{{"_id", "document-composite"}},
		},
		"FindProjectionExclusions": {
			filter: bson.D{{"_id", "document-composite"}},
			// TODO: https://github.com/FerretDB/FerretDB/issues/537
			projection: bson.D{{"foo", int32(0)}, {"array", false}},
			expected:   bson.D{{"_id", "document-composite"}, {"value", bson.D{{"foo", int32(42)}, {"42", "foo"}, {"array", bson.A{int32(42), "foo", nil}}}}},
		},
		"FindProjectionIDExclusion": {
			filter: bson.D{{"_id", "document-composite"}},
			// TODO: https://github.com/FerretDB/FerretDB/issues/537
			projection: bson.D{{"_id", false}, {"array", int32(1)}},
			expected:   bson.D{},
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cursor, err := collection.Find(ctx, tc.filter, options.Find().SetProjection(tc.projection))
			require.NoError(t, err)

			var actual []bson.D
			err = cursor.All(ctx, &actual)
			require.NoError(t, err)
			require.Len(t, actual, 1)
			AssertEqualDocuments(t, tc.expected, actual[0])
		})
	}
}

func TestQueryProjectionElemMatch(t *testing.T) {
	t.Parallel()
	providers := []shareddata.Provider{shareddata.Composites}
	ctx, collection := setup(t, providers...)

	_, err := collection.InsertMany(ctx, []any{
		bson.D{
			{"_id", "document-composite-2"},
			{"value", bson.A{
				bson.D{{"field", int32(42)}},
				bson.D{{"field", int32(44)}},
			}},
		},
	})
	require.NoError(t, err)

	for name, tc := range map[string]struct {
		projection  any
		expectedIDs []any
	}{
		"ElemMatch": {
			projection: bson.D{{
				"value",
				bson.D{{"$elemMatch", bson.D{{"field", bson.D{{"$eq", 42}}}}}},
			}},
			expectedIDs: []any{
				"document-composite-2",
			},
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cursor, err := collection.Find(
				ctx,
				bson.D{{"_id", "document-composite-2"}},
				options.Find().SetProjection(tc.projection),
				options.Find().SetSort(bson.D{{"_id", 1}}),
			)
			require.NoError(t, err)

			var actual []bson.D
			err = cursor.All(ctx, &actual)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedIDs, CollectIDs(t, actual))
		})
	}
}