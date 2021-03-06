package main

import (
	"path/filepath"
	"testing"
)

var (
	testdataTop = "testdata"
	testdata1   = filepath.Join(testdataTop, "dir1")
	testdata2   = filepath.Join(testdataTop, "dir2")
)

func sliceToSet(stuff []string) map[string]bool {
	set := make(map[string]bool)
	for _, thing := range stuff {
		set[thing] = true
	}
	return set
}

// CLI
// Error to have no arguments.
func TestCliRequiresOneOrMoreDirectories(t *testing.T) {
	emptySlice := make([]string, 0, 0)
	if validateArgs(emptySlice) == nil {
		t.Error("no arguments should be an error")
	}
}

// Error to specify something that doesn't exist.
func TestCliArgsMustExist(t *testing.T) {
	doesNotExist := []string{"Totally bogus file name"}
	if validateArgs(doesNotExist) == nil {
		t.Error("non-existent arguments should be an error")
	}
}

// Error to specify a file.
func TestCliArgsMustBeDirectories(t *testing.T) {
	file := []string{filepath.Join(testdata1, "intra-same")}
	if validateArgs(file) == nil {
		t.Error("files are not valid arguments")
	}
}

// Succeed if all arguments are directories.
func TestCliArgsDirectoriesOK(t *testing.T) {
	dir := []string{testdataTop, testdata1}
	if validateArgs(dir) != nil {
		t.Error("directories are acceptable")
	}
}

// Directory traversal
func TestSortDirContentsError(t *testing.T) {
	_, _, err := sortDirContents("bogus path")
	if err == nil {
		t.Fatal("expected an error")
	}
	file := filepath.Join(testdata1, "intra-same")
	_, _, err = sortDirContents(file)
	if err == nil {
		t.Fatal("expected an error")
	}
}

// Directory with only files.
func TestFindFiles(t *testing.T) {
	dir := []string{testdata2}
	files, err := findFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	fileSet := sliceToSet(files)
	if len(fileSet) != 2 {
		t.Fatalf("found %v files instead of 2", len(fileSet))
	}
	file1 := filepath.Join(testdata2, "inter-diff")
	file2 := filepath.Join(testdata2, "inter-same")
	for _, file := range []string{file1, file2} {
		if !fileSet[file] {
			t.Errorf("%v not found in %v", file1, fileSet)
		}
	}

	_, err = findFiles([]string{"bogus path"})
	if err == nil {
		t.Fatal("error expected")
	}
}

// Directory containing subdirectories.
func TestFindRecursively(t *testing.T) {
	dir := []string{testdataTop}
	files, err := findFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	fileSet := sliceToSet(files)
	if len(fileSet) != 8 {
		t.Fatalf("found %v files instead of 8", len(fileSet))
	}
	if !fileSet[filepath.Join(testdata1, "intra-diff1")] {
		t.Error("didn't find the intra-diff1 file")
	}
}

// Multiple directories.
func TestFindInMultipleDirectories(t *testing.T) {
	dirs := []string{testdata1, testdata2}
	files, err := findFiles(dirs)
	if err != nil {
		t.Fatal(err)
	}
	fileSet := sliceToSet(files)
	if len(fileSet) != 8 {
		t.Fatalf("found %v files instead of 8", len(fileSet))
	}
}

// Hashing
// Error out if the path is bogus.
func TestHashingUnknown(t *testing.T) {
	_, err := hashFile("bogus file name")
	if err == nil {
		t.Error("non-existent file didn't trigger an error")
	}
}

// Verify hashing works.
func TestHashing(t *testing.T) {
	intraSame1, err := hashFile(filepath.Join(testdata1, "intra-same1"))
	if err != nil {
		t.Fatal(err)
	}
	intraSame2, err := hashFile(filepath.Join(testdata1, "intra-same2"))
	if err != nil {
		t.Fatal(err)
	}
	if intraSame1 != intraSame2 {
		t.Errorf("%v != %v", intraSame1, intraSame2)
	}

	intraDiff1, err := hashFile(filepath.Join(testdata1, "intra-diff1"))
	if err != nil {
		t.Fatal(err)
	}
	intraDiff2, err := hashFile(filepath.Join(testdata1, "intra-diff2"))
	if err != nil {
		t.Fatal(err)
	}
	if intraDiff1 == intraDiff2 {
		t.Errorf("%v == %v", intraDiff1, intraDiff2)
	}
}

// Duplication detection.
func baseTestDuplicationDetectingDuplicates(findDupes func([]string) (HashToFiles, error), t *testing.T) {
	files := []string{filepath.Join(testdata1, "intra-same1"), filepath.Join(testdata1, "intra-same2")}
	dupes, err := findDupes(files)
	if err != nil {
		t.Fatal(err)
	}

	if len(dupes) != 1 {
		t.Fatalf("expected 1 set of dupes, not %v", len(dupes))
	}

	for _, given := range dupes {
		givenSet := sliceToSet(given)
		for _, path := range files {
			if !givenSet[path] {
				t.Fatalf("%v not in %v", path, given)
			}
		}
	}

	_, err = findDupes([]string{"bogus path"})
	if err == nil {
		t.Fatal("error expected")
	}
}

func TestDuplicationDetectingDuplicatesSync(t *testing.T) {
	baseTestDuplicationDetectingDuplicates(findDuplicates, t)
}

func baseTestDuplicationNoDupes(findDupes func([]string) (HashToFiles, error), t *testing.T) {
	files := []string{filepath.Join(testdata1, "intra-diff1"), filepath.Join(testdata1, "intra-diff2")}
	dupes, err := findDupes(files)
	if err != nil {
		t.Fatal(err)
	}

	if want, got := 2, len(dupes); want != got {
		t.Errorf("want %v, got %v (%v)", want, got, dupes)
	}
}

func TestDuplicationNoDupes(t *testing.T) {
	baseTestDuplicationNoDupes(findDuplicates, t)
}

// Concurrency

func TestHashFileAsyncError(t *testing.T) {
	// Buffer to prevent deadlock.
	response := make(chan MaybeHash, 1)
	hashFileAsync("bogus file path", response)
	result := <-response
	if result.err == nil {
		t.Fatal("expected an error")
	}
}

func TestDuplicationDetectingDuplicatesSyncAsync(t *testing.T) {
	baseTestDuplicationDetectingDuplicates(findDuplicatesConcurrently, t)
}

func TestDuplicationNoDupesAsync(t *testing.T) {
	baseTestDuplicationNoDupes(findDuplicatesConcurrently, t)
}
