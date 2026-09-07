package queries_test

import (
	"database/sql"
	"io"
	"log/slog"
	"testing"

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
			result, err := queries.GetTimeline(&types.FilterParams{OrderBy: "date", Direction: direction, PageSize: 1}, c)
			require.NoError(t, err)
			assert.Equal(t, 1, result.Meta.Total)
			require.Len(t, result.Photos, 1)
			require.Len(t, result.Timeline, 1)
			assert.Equal(t, 1, result.Timeline[0].Count)

			next, err := queries.GetTimeline(&types.FilterParams{OrderBy: "date", Direction: direction, PageSize: 1, Cursor: 1}, c)
			require.NoError(t, err)
			assert.Equal(t, 1, next.Meta.Total)
			assert.Empty(t, next.Photos)
			assert.Empty(t, next.Timeline)
		})
	}
}

func TestSearchTreatsOperatorsAndPunctuationLiterally(t *testing.T) {
	c := mediaFixture(t)
	for _, term := range []string{"NOT", "NIKON OR", "20250101-senično", `"NIKON"`, "(NIKON)"} {
		t.Run(term, func(t *testing.T) {
			result, err := queries.GetTimeline(&types.FilterParams{Term: term, OrderBy: "date", Direction: "desc"}, c)
			require.NoError(t, err)
			if term == "NOT" || term == "20250101-senično" {
				assert.Equal(t, 1, result.Meta.Total)
			}
		})
	}
}
