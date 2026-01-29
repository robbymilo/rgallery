package queries

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/robbymilo/rgallery/pkg/database"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type MediaCollection struct {
	Items []Media
}

// GetMemories returns media items that occurred on the today's date in previous years and groups them into years.
func GetMemories(c Conf) (Days, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 5,
	})
	if err != nil {
		return nil, fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer pool.Close()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	total := 20
	today := time.Now()
	var dates []string

	for i := 1; i <= total; i++ {
		pastDate := today.AddDate(-i, 0, 0).Format("2006-01-02")
		dates = append(dates, fmt.Sprintf("DATE('%s')", pastDate))
	}

	query := fmt.Sprintf(`
SELECT %s
FROM media
WHERE DATE(date) IN (%s)
`, database.Columns(), strings.Join(dates, ", "))

	stmt, err := conn.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("error preparing SELECT statement: %v", err)
	}

	result, err := parseMediaRows(stmt, c)
	if err != nil {
		return nil, err
	}

	mediaCollection := MediaCollection{Items: result}

	sortedDays, err := mediaCollection.sortByYear()
	if err != nil {
		return nil, fmt.Errorf("error sorting memories: %v", err)
	}

	err = stmt.Finalize()
	if err != nil {
		return nil, fmt.Errorf("error finalizing statement: %v", err)
	}

	return sortedDays, nil
}

func (mediaSorter *MediaCollection) sortByYear() (Days, error) {
	// group by year
	group := make(map[int][]Media)
	for _, image := range mediaSorter.Items {
		// Always use UTC time for grouping and output
		utcDate := image.Date.UTC()
		image.Date = utcDate
		year := utcDate.Year()
		group[year] = append(group[year], image)
	}

	// sort by year
	years := []Day{}
	for itemYear, yearMediaItems := range group {
		if len(yearMediaItems) == 0 {
			continue
		}
		// Use UTC date for root value
		referenceDate := yearMediaItems[0].Date.UTC()
		ago := time.Now().UTC().Year() - itemYear
		var final []Media
		for i := range yearMediaItems {
			if i < 3 {
				final = append(final, yearMediaItems[i])
			}
		}

		// handle when date occurs on today but in the future
		if ago < 1 {
			ago = 1
		}

		years = append(
			years,
			Day{
				Key:   ago,
				Value: referenceDate.Format("2006-01-02"),
				Media: final,
				Total: len(yearMediaItems),
			})
	}

	sort.SliceStable(years, func(i, j int) bool {
		return years[i].Value > years[j].Value
	})

	return years, nil
}
