package queries

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/sizes"
	"github.com/robbymilo/rgallery/pkg/types"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type Folder = types.Folder
type FolderMedia = types.FolderMedia
type Directory = types.Directory
type TreeNode = types.TreeNode

// GetFolders returns a list of folders organized in a tree structure.
func GetFolders(params FilterParams, group string, pageSize, offset int, c Conf) ([]*TreeNode, error) {
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

	query := fmt.Sprintf(`
      WITH FolderImageCounts AS (
        SELECT
          folder,
          COUNT(*) as total_images
        FROM media
        GROUP BY folder
      ), RankedMedia AS (
        SELECT
          m.folder, m.hash, m.path, m.width, m.height, m.color, m.date,
          ROW_NUMBER() OVER (PARTITION BY m.folder ORDER BY m.date DESC) as row_num
        FROM media m
      )
      SELECT
          f.id,
          f.key,
          GROUP_CONCAT(CASE WHEN rm.row_num <= 5 THEN rm.hash ELSE NULL END) as hashes,
          GROUP_CONCAT(CASE WHEN rm.row_num <= 5 THEN rm.path ELSE NULL END) as paths,
          GROUP_CONCAT(CASE WHEN rm.row_num <= 5 THEN rm.width ELSE NULL END) as widths,
          GROUP_CONCAT(CASE WHEN rm.row_num <= 5 THEN rm.height ELSE NULL END) as heights,
          GROUP_CONCAT(CASE WHEN rm.row_num <= 5 THEN rm.color ELSE NULL END) as colors,
          COALESCE(fic.total_images, 0) as image_count,
          GROUP_CONCAT(CASE WHEN rm.row_num <= 5 THEN rm.date ELSE NULL END) as dates
      FROM folders f
      LEFT JOIN FolderImageCounts fic ON f.key = fic.folder
      LEFT JOIN RankedMedia rm ON f.key = rm.folder AND rm.row_num <= 5
      GROUP BY f.id, f.key, COALESCE(fic.total_images, 0)
      ORDER BY f.key %s
      LIMIT %d OFFSET %d`, params.Direction, pageSize, offset)

	stmt, err := conn.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("error preparing SELECT statement: %v", err)
	}

	dirMap := make(map[string]*Directory)

	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("error stepping through result set: %v", err)
		}
		if !hasRow {
			break
		}

		// Parse concatenated values
		hashes := strings.Split(stmt.ColumnText(2), ",")
		paths := strings.Split(stmt.ColumnText(3), ",")
		widths := strings.Split(stmt.ColumnText(4), ",")
		heights := strings.Split(stmt.ColumnText(5), ",")
		colors := strings.Split(stmt.ColumnText(6), ",")
		imageCount := stmt.ColumnInt(7)

		var mediaList []FolderMedia
		for i := 0; i < len(hashes) && i < 5 && hashes[0] != ""; i++ {
			width, _ := strconv.Atoi(widths[i])
			height, _ := strconv.Atoi(heights[i])
			hash, _ := strconv.ParseUint(hashes[i], 10, 32)

			media := FolderMedia{
				Hash:   uint32(hash),
				Path:   paths[i],
				Width:  width,
				Height: height,
				Color:  colors[i],
			}
			media.Srcset = sizes.Srcset(media.Hash, media.Width, media.Path, c)
			mediaList = append(mediaList, media)
		}

		id := stmt.ColumnInt(0)
		key := stmt.ColumnText(1)
		dir := &Directory{
			Id:         id,
			Key:        key,
			Media:      nil,
			ImageCount: imageCount,
		}

		if len(mediaList) > 0 {
			dir.Media = &mediaList
		}

		dirMap[key] = dir
	}

	// After building dirMap, replace the sorting and return code with:
	var dirs []Directory
	for _, dir := range dirMap {
		dirs = append(dirs, *dir)
	}

	err = stmt.Finalize()
	if err != nil {
		return nil, fmt.Errorf("error finalizing statement: %v", err)
	}

	return buildTree(dirs), nil
}

// buildTree converts a flat directory slice into a hierarchical tree structure
func buildTree(dirs []Directory) []*TreeNode {
	root := make(map[string]*TreeNode)

	// First pass: create all nodes
	for _, dir := range dirs {
		parts := strings.Split(dir.Key, "/")
		currentPath := ""

		// Create or update nodes for each path segment
		for i, part := range parts {
			if i > 0 {
				currentPath += "/"
			}
			currentPath += part

			if _, exists := root[currentPath]; !exists {
				root[currentPath] = &TreeNode{
					Name:     part,
					Path:     currentPath,
					Children: []*TreeNode{},
				}
			}

			// If this is the full path, add the media info
			if currentPath == dir.Key {
				root[currentPath].Id = dir.Id
				root[currentPath].Media = dir.Media
				root[currentPath].ImageCount = dir.ImageCount
			}
		}
	}

	// Second pass: build parent-child relationships
	for path, node := range root {
		parentPath := filepath.Dir(path)
		if parentPath != "." && parentPath != path {
			if parent, exists := root[parentPath]; exists {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	// Create final result with only root level nodes
	var result []*TreeNode
	for path, node := range root {
		if !strings.Contains(path[1:], "/") {
			result = append(result, node)
		}
	}

	// Sort children recursively
	var sortTree func([]*TreeNode)
	sortTree = func(nodes []*TreeNode) {
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].Path < nodes[j].Path
		})
		for _, node := range nodes {
			sortTree(node.Children)
		}
	}
	sortTree(result)

	return result
}

// GetTotalFolders returns the total of all folders.
func GetTotalFolders(c Conf) (int, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 1,
	})
	if err != nil {
		return 0, fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			c.Logger.Error("pool.Close error:", "err", err)
		}
	}()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return 0, fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	query := `SELECT COUNT(*) FROM folders`
	stmt, err := conn.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf("error preparing query: %v", err)
	}
	defer func() {
		if finalizeErr := stmt.Finalize(); finalizeErr != nil {
			err = fmt.Errorf("error finalizing statement: %v", finalizeErr)
		}
	}()

	var total int
	hasRow, err := stmt.Step()
	if err != nil {
		return 0, fmt.Errorf("error executing query: %v", err)
	}
	if hasRow {
		total = int(stmt.ColumnInt64(0))
	}

	return total, nil
}
