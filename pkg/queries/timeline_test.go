package queries_test

import (
	"database/sql"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mediaFixture(t *testing.T) types.Conf {
	t.Helper()
	c := types.Conf{Data: t.TempDir(), Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	database.CreateDB(c)
	db, err := sql.Open("sqlite", database.NewSqlConnectionString(c))
	require.NoError(t, err)
	defer db.Close()
	for _, id := range []int{1, 2, 3} {
		_, err = db.Exec(`INSERT INTO media (hash, path, date, modified, folder, rating, width, height, latitude, longitude) VALUES (?, '20250101-senično/NOT-NIKON.jpg', '2025-01-01T00:00:00.000Z', '2025-01-01T00:00:00.000Z', '20250101-senično', 5, 100, 100, 46, 14)`, id)
		require.NoError(t, err)
	}
	return c
}

func TestTimelineKeepsTimestampCollisionsAcrossPages(t *testing.T) {
	c := mediaFixture(t)
	for _, direction := range []string{"asc", "desc"} {
		t.Run(direction, func(t *testing.T) {
			var ids []uint32
			for offset := 0; offset < 3; offset++ {
				result, err := queries.GetTimeline(&types.FilterParams{OrderBy: "date", Direction: direction, PageSize: 1, Cursor: offset}, c)
				require.NoError(t, err)
				assert.Equal(t, 3, result.Meta.Total)
				require.Len(t, result.Photos, 1)
				ids = append(ids, result.Photos[0].Id)
				if offset == 0 {
					require.Len(t, result.Timeline, 1)
					assert.Equal(t, 3, result.Timeline[0].Count)
				}
			}
			if direction == "asc" {
				assert.Equal(t, []uint32{1, 2, 3}, ids)
			} else {
				assert.Equal(t, []uint32{3, 2, 1}, ids)
			}
		})
	}
	date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	previous, err := queries.GetPrevious(date, 2, types.FilterParams{}, c)
	require.NoError(t, err)
	require.Len(t, previous, 1)
	assert.Equal(t, uint32(3), previous[0].Hash)
	next, err := queries.GetNext(date, 2, 3, types.FilterParams{}, previous, c)
	require.NoError(t, err)
	require.Len(t, next, 1)
	assert.Equal(t, uint32(1), next[0].Hash)
	items, err := queries.GetMapItems(c)
	require.NoError(t, err)
	assert.Len(t, items, 3)
	count, err := queries.GetTotalOfFolder("folder", "20250101-senično", c)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestSearchTreatsOperatorsAndPunctuationLiterally(t *testing.T) {
	c := mediaFixture(t)
	for _, term := range []string{"NOT", "NIKON OR", "20250101-senično", `"NIKON"`, "(NIKON)"} {
		t.Run(term, func(t *testing.T) {
			result, err := queries.GetTimeline(&types.FilterParams{Term: term, OrderBy: "date", Direction: "desc"}, c)
			require.NoError(t, err)
			if term == "NOT" || term == "20250101-senično" {
				assert.Equal(t, 3, result.Meta.Total)
			}
		})
	}
}
