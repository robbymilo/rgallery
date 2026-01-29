package queries

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/robbymilo/rgallery/pkg/database"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// GetGear returns the total unique counts for a specified gear column.
func GetGear(column string, c Conf) (GearItems, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			c.Logger.Error("pool.Close error:", "err", err)
		}
	}()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	query := fmt.Sprintf(
		`SELECT %s, count(%s) AS total
			 FROM media
			 WHERE date != ?
			 GROUP BY %s
			 ORDER BY total DESC`, column, column, column)

	stmt, err := conn.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("error preparing query: %v", err)
	}
	defer func() {
		if finalizeErr := stmt.Finalize(); finalizeErr != nil {
			err = fmt.Errorf("error finalizing statement: %v", finalizeErr)
		}
	}()

	stmt.BindText(1, "0001-01-01T00:00:00.000Z")

	var gear GearItems
	lensMap := make(map[string]int)
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("error stepping through query results: %v", err)
		}
		if !hasRow {
			break
		}

		name := stmt.ColumnText(0)
		if column == "focallength35" {
			name = strings.TrimSuffix(name, ".0")
		}
		total := int(stmt.ColumnInt64(1))

		if name != "" && total != 0 {
			if column == "lens" {
				lensMap[name] = total
			} else {
				gear = append(gear, GearItem{
					Name:  name,
					Total: total,
				})
			}
		}
	}

	if column == "lens" {
		lensMapFinal := make(map[string]int)
		for k, v := range lensMap {
			n := c.Aliases.Lenses[strings.TrimSpace(k)]
			if n != "" {
				lensMapFinal[n] = lensMapFinal[n] + v
			} else {
				lensMapFinal[k] = v
			}
		}
		var gearLens GearItems
		for lens, total := range lensMapFinal {
			gearLens = append(gearLens, GearItem{
				Name:  lens,
				Total: total,
			})
		}

		sort.Slice(gearLens, func(i, j int) bool {
			if gearLens[i].Total != gearLens[j].Total {
				return gearLens[i].Total > gearLens[j].Total // sort by Total descending
			}
			return gearLens[i].Name < gearLens[j].Name // sort by Name ascending
		})

		gear = gearLens

	}

	return gear, nil

}
