package datastore

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDatabasePut(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "db-testing")
	if err != nil {
		t.Fatal("Failed to create temporary directory:", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewDb(tempDir, 150)
	if err != nil {
		t.Fatal("Failed to create new database:", err)
	}
	defer db.Close()

	testPairs := [][]string{
		{"key1", "value1"},
		{"key2", "value2"},
		{"key3", "value3"},
	}

	outputFile, err := os.Open(filepath.Join(tempDir, outFileName+"0"))
	if err != nil {
		t.Fatal("Failed to open output file:", err)
	}
	defer outputFile.Close()

	t.Run("Put and get", func(t *testing.T) {
		for _, pair := range testPairs {
			if err := db.Put(pair[0], pair[1]); err != nil {
				t.Errorf("Put operation failed for key %s: %v", pair[0], err)
			}
			retrievedValue, err := db.Get(pair[0])
			if err != nil {
				t.Errorf("Get operation failed for key %s: %v", pair[0], err)
			}
			if retrievedValue != pair[1] {
				t.Errorf("Value mismatch for key %s: expected %s, got %s", pair[0], pair[1], retrievedValue)
			}
		}
	})

	initialFileInfo, err := outputFile.Stat()
	if err != nil {
		t.Fatal("Failed to get file information:", err)
	}
	initialSize := initialFileInfo.Size()

	t.Run("File growth", func(t *testing.T) {
		for _, pair := range testPairs {
			if err := db.Put(pair[0], pair[1]); err != nil {
				t.Errorf("Put operation failed for key %s: %v", pair[0], err)
			}
		}
		currentFileInfo, err := outputFile.Stat()
		if err != nil {
			t.Fatal("Failed to get file information:", err)
		}
		expectedSize := initialSize * 2
		if currentFileInfo.Size() != expectedSize {
			t.Errorf("File size mismatch: expected %d, got %d", expectedSize, currentFileInfo.Size())
		}
	})

	t.Run("New database process", func(t *testing.T) {
		if err := db.Close(); err != nil {
			t.Fatal("Failed to close database:", err)
		}
		db, err = NewDb(tempDir, 150)
		if err != nil {
			t.Fatal("Failed to reopen database:", err)
		}

		for _, pair := range testPairs {
			retrievedValue, err := db.Get(pair[0])
			if err != nil {
				t.Errorf("Get operation failed for key %s after reopening database: %v", pair[0], err)
			}
			if retrievedValue != pair[1] {
				t.Errorf("Value mismatch for key %s after reopening database: expected %s, got %s", pair[0], pair[1], retrievedValue)
			}
		}
	})
}

func TestDatabaseSegmentation(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "db-testing")
	if err != nil {
		t.Fatal("Failed to create temporary directory:", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewDb(tempDir, 45)
	if err != nil {
		t.Fatal("Failed to create new database:", err)
	}
	defer db.Close()

	t.Run("Create new file on segmentation", func(t *testing.T) {
		db.Put("key1", "value1")
		db.Put("key2", "value2")
		db.Put("key3", "value3")
		db.Put("key2", "value5")

		expectedSegments := 2
		if len(db.segments.list) != expectedSegments {
			t.Errorf("Segmentation error: expected %d segments, got %d", expectedSegments, len(db.segments.list))
		}
	})

	t.Run("Start segmentation", func(t *testing.T) {
		db.Put("key4", "value4")

		expectedSegments := 3
		if len(db.segments.list) != expectedSegments {
			t.Errorf("Segmentation error: expected %d segments, got %d", expectedSegments, len(db.segments.list))
		}

		time.Sleep(2 * time.Second)

		expectedSegmentsAfterSleep := 2
		if len(db.segments.list) != expectedSegmentsAfterSleep {
			t.Errorf("Segmentation error after sleep: expected %d segments, got %d", expectedSegmentsAfterSleep, len(db.segments.list))
		}
	})

	t.Run("Does not store duplicates", func(t *testing.T) {
		file, err := os.Open(db.segments.list[0].path)
		if err != nil {
			t.Fatal("Failed to open segment file:", err)
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			t.Fatal("Failed to get segment file information:", err)
		}
		expectedSize := int64(66)
		if fileInfo.Size() != expectedSize {
			t.Errorf("Segment file size mismatch: expected %d, got %d", expectedSize, fileInfo.Size())
		}
	})

	t.Run("Does not store new values of duplicate keys", func(t *testing.T) {
		expectedValue := "value5"
		retrievedValue, err := db.Get("key2")
		if err != nil {
			t.Fatal("Failed to get value for key 'key2':", err)
		}
		if retrievedValue != expectedValue {
			t.Errorf("Value mismatch for key 'key2': expected %s, got %s", expectedValue, retrievedValue)
		}
	})
}
