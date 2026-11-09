package rgallery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/scanner"
	"github.com/stretchr/testify/require"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type scanArtifact struct {
	Path   string `json:"path"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type scanSnapshot struct {
	Status     string                      `json:"status"`
	Tables     map[string][]map[string]any `json:"tables"`
	Artifacts  []scanArtifact              `json:"artifacts"`
	ScanErrors []map[string]any            `json:"scan_errors"`
}

func newHermeticTestConf(t testing.TB) Conf {
	t.Helper()
	root := t.TempDir()
	media := filepath.Join(root, "media")
	copyTree(t, filepath.Join("..", "..", "testdata", "media"), media)

	return Conf{
		DisableAuth:      true,
		Media:            media,
		Cache:            filepath.Join(root, "cache"),
		Data:             filepath.Join(root, "data"),
		PreGenerateThumb: true,
		LocationService:  "",
		LocationDataset:  "Provinces10",
		Logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		Memories:         false,
		TileServer:       `/api/tiles/{z}/{x}/{y}.png`,
	}
}

// BenchmarkScanBaselines records cold-cache and warm-cache scan metrics for
// the characterized fixture corpus. Run with -benchtime=1x when collecting a
// baseline; a full scan intentionally remains expensive and representative.
func BenchmarkScanBaselines(b *testing.B) {
	b.Run("cold", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			c := newHermeticTestConf(b)
			database.CreateDB(c)
			b.StartTimer()
			if _, err := scanner.Scan("default", c, ca); err != nil {
				b.Fatal(err)
			}
			b.StopTimer()
			reportScanMetrics(b, scanner.LastScanMetrics())
			b.StartTimer()
		}
	})

	b.Run("warm", func(b *testing.B) {
		b.StopTimer()
		c := newHermeticTestConf(b)
		database.CreateDB(c)
		if _, err := scanner.Scan("default", c, ca); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			if _, err := scanner.Scan("default", c, ca); err != nil {
				b.Fatal(err)
			}
			b.StopTimer()
			reportScanMetrics(b, scanner.LastScanMetrics())
			b.StartTimer()
		}
	})
}

func reportScanMetrics(b *testing.B, m scanner.ScanMetrics) {
	b.ReportMetric(float64(m.Duration.Microseconds()), "scan-us")
	b.ReportMetric(float64(m.FilesDiscovered), "files/scan")
	b.ReportMetric(float64(m.StatCalls), "stats/scan")
	b.ReportMetric(float64(m.ExternalProcessLaunches), "external-processes/scan")
	b.ReportMetric(float64(m.DatabaseTransactions), "db-transactions/scan")
	b.ReportMetric(float64(m.CacheWrites), "cache-writes/scan")
	b.ReportMetric(float64(m.PeakMemoryBytes), "peak-alloc-bytes/scan")
	b.ReportMetric(m.ThroughputFilesPerSecond, "files/s")
	b.ReportMetric(float64(m.FilesystemDiscovery.Microseconds()), "discovery-us/scan")
	b.ReportMetric(float64(m.RawMetadata.Microseconds()), "metadata-us/scan")
	b.ReportMetric(float64(m.ConvertedLens.Microseconds()), "lens-us/scan")
	b.ReportMetric(float64(m.ImageProcessing.Microseconds()), "image-processing-us/scan")
	b.ReportMetric(float64(m.ThumbnailGeneration.Microseconds()), "thumbnails-us/scan")
	b.ReportMetric(float64(m.ReverseGeocoding.Microseconds()), "geocoding-us/scan")
	b.ReportMetric(float64(m.DatabaseWrites.Microseconds()), "database-us/scan")
}

func TestScanCharacterization(t *testing.T) {
	c = newHermeticTestConf(t)
	database.CreateDB(c)

	firstStatus, err := scanner.Scan("default", c, ca)
	require.NoError(t, err)
	require.Contains(t, firstStatus, "Scan complete")
	metrics := scanner.LastScanMetrics()
	require.Equal(t, "default", metrics.ScanType)
	require.Positive(t, metrics.FilesDiscovered)
	require.Positive(t, metrics.StatCalls)
	require.Positive(t, metrics.RawMetadataFiles)
	require.Positive(t, metrics.ConvertedLensFiles)
	require.Positive(t, metrics.ImageProcessingFiles)
	require.Positive(t, metrics.ThumbnailCount)
	require.Positive(t, metrics.DatabaseTransactions)
	require.Positive(t, metrics.CacheWrites)
	require.Positive(t, metrics.ExternalProcessLaunches)
	require.Equal(t, 2, metrics.ExternalProcessLaunches, "one raw and one converted ExifTool instance per scan")
	require.Positive(t, metrics.PeakMemoryBytes)
	first := makeScanSnapshot(t, c, firstStatus)

	secondStatus, err := scanner.Scan("default", c, ca)
	require.NoError(t, err)
	require.Contains(t, secondStatus, "Scan complete")
	require.Equal(t, 2, scanner.LastScanMetrics().ExternalProcessLaunches, "scan-owned ExifTool instances should be reused even when no files need metadata")
	second := makeScanSnapshot(t, c, secondStatus)
	second.Status = first.Status
	require.Equal(t, first, second, "an unchanged second scan must be idempotent")

	assertDuplicatePair(t, c, []string{
		"2017/20170624-idaho/20170624-sawtooth-mountain-biking-robbymilo-0030.jpg",
		"2017/20170624-idaho/20170624-sawtooth-mountain-biking-robbymilo-0030 copy.jpg",
	})
	assertDuplicatePair(t, c, []string{
		"2018/20180304-bogus/20180304-mores-mountain-ski-robbymilo-0230.jpg",
		"2018/20180304-bogus/20180304-mores-mountain-ski-robbymilo-0230 2.jpg",
	})

	goldenPath := filepath.Join("..", "..", "testdata", "scan-snapshot.json")
	encoded, err := json.MarshalIndent(first, "", "  ")
	require.NoError(t, err)
	encoded = append(encoded, '\n')
	if os.Getenv("UPDATE_SCAN_SNAPSHOT") == "1" {
		require.NoError(t, os.WriteFile(goldenPath, encoded, 0o644))
	}
	want, err := os.ReadFile(goldenPath)
	require.NoError(t, err)
	require.JSONEq(t, string(want), string(encoded))
}

func TestScanLifecycleCharacterization(t *testing.T) {
	c = newHermeticTestConf(t)
	database.CreateDB(c)
	require.NoError(t, scanAndCheck(t, c, "default"))
	initial := mediaCount(t, c)

	for _, scanType := range []string{"metadata", "deep"} {
		require.NoError(t, scanAndCheck(t, c, scanType))
		require.Equal(t, initial, mediaCount(t, c), "%s scan changed the media set", scanType)
	}
	status, err := scanner.Scan("thumb", c, ca)
	require.NoError(t, err)
	require.Empty(t, status, "the default scanner currently ignores the thumb scan type")
	require.Equal(t, initial, mediaCount(t, c), "thumbnail scan changed the media set")

	added := filepath.Join(c.Media, "misc", "phase1-added.jpg")
	require.NoError(t, copyFile(filepath.Join(c.Media, "misc", "105-2.jpg"), added))
	require.NoError(t, scanAndCheck(t, c, "default"))
	require.Equal(t, initial+1, mediaCount(t, c), "adding a file should add one media row")

	updated := filepath.Join(c.Media, "misc", "105-3.jpg")
	require.NoError(t, os.Chtimes(updated, time.Now().Add(time.Second), time.Now().Add(time.Second)))
	require.NoError(t, scanAndCheck(t, c, "default"))
	require.Equal(t, initial+1, mediaCount(t, c), "modifying a file should not add a row")

	require.NoError(t, os.Remove(added))
	require.NoError(t, scanAndCheck(t, c, "default"))
	require.Equal(t, initial, mediaCount(t, c), "deleting a file should remove its media row")
}

func scanAndCheck(t *testing.T, c Conf, scanType string) error {
	t.Helper()
	_, err := scanner.Scan(scanType, c, ca)
	return err
}

func mediaCount(t *testing.T, c Conf) int {
	t.Helper()
	conn, err := sqlite.OpenConn(database.NewSqlConnectionString(c), sqlite.OpenReadOnly)
	require.NoError(t, err)
	defer func() { require.NoError(t, conn.Close()) }()
	var count int
	require.NoError(t, sqlitex.ExecuteTransient(conn, "SELECT COUNT(*) FROM media", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			count = stmt.ColumnInt(0)
			return nil
		},
	}))
	return count
}

func makeScanSnapshot(t *testing.T, c Conf, status string) scanSnapshot {
	t.Helper()
	conn, err := sqlite.OpenConn(database.NewSqlConnectionString(c), sqlite.OpenReadOnly)
	require.NoError(t, err)
	defer func() { require.NoError(t, conn.Close()) }()

	tables := map[string][]map[string]any{}
	for name, query := range map[string]string{
		"media":          `SELECT hash,path,subject,width,height,ratio,padding,date,modified,folder,rating,shutterspeed,aperture,iso,lens,camera,focallength,altitude,latitude,longitude,mediatype,focusdistance,focallength35,color,location,description,title,software,offset,rotation FROM media ORDER BY hash`,
		"folders":        `SELECT id,key FROM folders ORDER BY id`,
		"tags":           `SELECT id,key,value FROM tags ORDER BY id`,
		"images_tags":    `SELECT image_id,tag_id FROM images_tags ORDER BY image_id,tag_id`,
		"scan_errors":    `SELECT path,modified,error FROM scan_errors ORDER BY path`,
		"images_virtual": `SELECT hash,path,subject,date,modified,folder,mediatype FROM images_virtual ORDER BY hash`,
	} {
		tables[name] = queryRows(t, conn, query)
	}

	artifacts := []scanArtifact{}
	require.NoError(t, filepath.WalkDir(c.Cache, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(c.Cache, path)
		require.NoError(t, err)
		artifact := scanArtifact{Path: filepath.ToSlash(relative)}
		if strings.HasSuffix(strings.ToLower(path), ".jpg") || strings.HasSuffix(strings.ToLower(path), ".png") {
			if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
				artifact.Width, artifact.Height = cfg.Width, cfg.Height
			}
		}
		artifacts = append(artifacts, artifact)
		return nil
	}))
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })

	return scanSnapshot{Status: scanStatus(status), Tables: tables, Artifacts: artifacts, ScanErrors: tables["scan_errors"]}
}

func queryRows(t *testing.T, conn *sqlite.Conn, query string) []map[string]any {
	t.Helper()
	result := []map[string]any{}
	require.NoError(t, sqlitex.ExecuteTransient(conn, query, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			row := map[string]any{}
			for i := 0; i < stmt.ColumnCount(); i++ {
				switch stmt.ColumnType(i) {
				case sqlite.TypeInteger:
					row[stmt.ColumnName(i)] = stmt.ColumnInt64(i)
				case sqlite.TypeFloat:
					row[stmt.ColumnName(i)] = stmt.ColumnFloat(i)
				case sqlite.TypeText:
					row[stmt.ColumnName(i)] = stmt.ColumnText(i)
				case sqlite.TypeBlob:
					data := make([]byte, stmt.ColumnLen(i))
					stmt.ColumnBytes(i, data)
					row[stmt.ColumnName(i)] = string(data)
				case sqlite.TypeNull:
					row[stmt.ColumnName(i)] = nil
				}
			}
			result = append(result, row)
			return nil
		},
	}))
	return result
}

func assertDuplicatePair(t *testing.T, c Conf, paths []string) {
	t.Helper()
	conn, err := sqlite.OpenConn(database.NewSqlConnectionString(c), sqlite.OpenReadOnly)
	require.NoError(t, err)
	defer func() { require.NoError(t, conn.Close()) }()
	for _, path := range paths {
		var count int
		require.NoError(t, sqlitex.ExecuteTransient(conn, "SELECT COUNT(*) FROM media WHERE path = ?", &sqlitex.ExecOptions{
			Args: []any{path},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				count = stmt.ColumnInt(0)
				return nil
			},
		}))
		require.Equal(t, 1, count, "duplicate fixture must remain a separate media row: %s", path)
	}
	var groups int
	require.NoError(t, sqlitex.ExecuteTransient(conn, "SELECT COUNT(DISTINCT date) FROM media WHERE path IN (?, ?)", &sqlitex.ExecOptions{
		Args: []any{paths[0], paths[1]},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			groups = stmt.ColumnInt(0)
			return nil
		},
	}))
	require.Equal(t, 1, groups, "duplicate fixture should retain timestamp grouping")
}

func scanStatus(status string) string {
	if strings.Contains(status, "Scan complete") {
		return "complete"
	}
	return status
}

func copyTree(t testing.TB, source, destination string) {
	t.Helper()
	require.NoError(t, filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
		return os.Chtimes(target, time.Date(2023, 11, 21, 20, 44, 53, 923000000, time.UTC), time.Date(2023, 11, 21, 20, 44, 53, 923000000, time.UTC))
	}))
}

func copyFile(source, destination string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(destination, data, 0o644)
}
