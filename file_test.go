package main

import (
  "os"
  "time"
  "strings"
  "testing"
  "io/ioutil"
  "path/filepath"
)

func TestCheckDirInvalid(t *testing.T) {
  tmpFile(t, "", func(in *os.File){
    _, err := checkDir(in.Name())
    if err == nil {
      t.Errorf("Expected error, got nil")
    }
  })
}

func TestCheckDir(t *testing.T) {
  td, err := ioutil.TempDir("", "")
  if err != nil {
    t.Fatal(err)
  }
  defer os.RemoveAll(td)

  _, err = checkDir(td)
  if err != nil {
    t.Errorf("Expected nil, got %v", err)
  }
}

func TestStringSliceFromFileNotExist(t *testing.T) {
  _, err := stringSliceFromFile(".ssync-this-file-does-not-exist")
  if err == nil {
    t.Errorf("Expected file not found error, got nil")
  }
}

func TestStringSliceFromFile(t *testing.T) {
  sliceTests := []map[string][]string{
    { "hello\nworld\n!\n": { "!", "hello", "world" } },
    { "\n\n  \n0 \n": { "0" } },
  }

  for i := range sliceTests {
    for k, v := range sliceTests[i] {
      tmpFile(t, k, func(in *os.File){
        r, _ := stringSliceFromFile(in.Name())
        if strings.Join(r, "\n") != strings.Join(v, "\n") {
          t.Errorf("Expected %v, got %v", v, r)
        }
      })
    }
  }
}

func TestStringSliceFromPathWalk(t *testing.T) {
  result := []string{
    ".ssync-test",
    "dir1",
    "dir1/dir2",
    "dir1/dir2/file3",
    "dir1/file2",
    "file1",
  }

  dir, _ := createTestFiles(t, testFiles)
  defer os.RemoveAll(dir)

  paths, err := stringSliceFromPathWalk(dir)
  if err != nil {
    t.Fatal(err)
  }

  if strings.Join(paths, "\n") != strings.Join(result, "\n") {
    t.Errorf("Expected %v, got %v", result, paths)
  }
}

// TestDeleteConfirm & TestDelete also fulfill testing of pathsThatExist

func TestDeleteConfirm(t *testing.T) {
  dir, _ := createTestFiles(t, testFiles)
  defer os.RemoveAll(dir)

  removes := []string{
    "file1",
    "dir1/dir2",
    "extra-does-not-exist-path",
  }

  delTests := map[string]bool{
    "Y": true,
    "N": false,
    "": false,
  }

  for input, result := range delTests {
    tmpFile(t, input, func(in *os.File){
      r := deleteConfirm(removes, dir, in)
      if r != result {
        t.Errorf("Expected %v, got %v", result, r)
      }
    })
  }
}

func TestDelete(t *testing.T){
  dir, _ := createTestFiles(t, testFiles)
  defer os.RemoveAll(dir)

  removes := []string{
    "file1",
    "dir1/dir2",
  }

  delete(removes, dir)

  for _, v := range removes {
    fullpath := filepath.Join(dir, filepath.Join(strings.Split(v, "/")...))
    if _, err := os.Stat(fullpath); err == nil {
      t.Errorf("Expected '%v' to be deleted", v)
    }
  }
}

type testCopyAllFunc func(in, out string, ip, op []string)

func testCopyAll(t *testing.T, f testCopyAllFunc){
  srcPath, _ := createTestFiles(t, testFiles)
  defer os.RemoveAll(srcPath)

  srcPaths, err := stringSliceFromPathWalk(srcPath)
  if err != nil {
    t.Fatal(err)
  }

  destPath, err := ioutil.TempDir("", "")
  if err != nil {
    t.Fatal(err)
  }
  defer os.RemoveAll(destPath)

  err = copyAll(srcPaths, srcPath, destPath)
  if err != nil {
    t.Fatal(err)
  }

  destPaths, err := stringSliceFromPathWalk(destPath)
  if err != nil {
    t.Fatal(err)
  }

  f(srcPath, destPath, srcPaths, destPaths)
}

func TestCopyAll(t *testing.T){
  testCopyAll(t, func(srcPath, destPath string, srcPaths, destPaths []string){
    // ensure all srcPaths equal destPaths
    if strings.Join(srcPaths, "\n") != strings.Join(destPaths, "\n") {
      t.Errorf("Expected %v, got %v", srcPaths, destPaths)
    }

    // ensure specified modified timestamp was set
    modTime, _ := time.Parse("2006-01-02", testFiles[1].Date)
    destFullpath := filepath.Join(destPath, "file1")
    fi, _ := os.Stat(destFullpath)
    if fi.ModTime().UTC() != modTime {
      t.Errorf("Expected %v, got %v", modTime, fi.ModTime().UTC())
    }
  })
}

func TestMostRecentlyModified(t *testing.T){
  testCopyAll(t, func(srcPath, destPath string, srcPaths, destPaths []string){

    // ensure modified timestamp is preserved in copyFile
    a, b, _ := mostRecentlyModified("file1", srcPath, destPath)
    if a != "" || b != "" {
      t.Errorf("Expected equal timestamps")
    }

    // ensure blank when checking directory
    a, b, _ = mostRecentlyModified("dir1", srcPath, destPath)
    if a != "" || b != "" {
      t.Errorf("Expected blank timestamps for directory")
    }

    ct := time.Now().Local()

    // ensure when srcPath most recently modified
    srcFullpath := filepath.Join(srcPath, "dir1/file2")
    if err := os.Chtimes(srcFullpath, ct, ct); err != nil {
      t.Fatal(err)
    }

    a, b, _ = mostRecentlyModified("dir1/file2", srcPath, destPath)
    if a != srcPath {
      t.Errorf("Expected %v, got %v", srcPath, a)
    }

    // ensure when destPath most recently modified
    destFullpath := filepath.Join(destPath, "dir1/dir2/file3")
    if err := os.Chtimes(destFullpath, ct, ct); err != nil {
      t.Fatal(err)
    }

    a, b, _ = mostRecentlyModified("dir1/dir2/file3", srcPath, destPath)
    if a != destPath {
      t.Errorf("Expected %v, got %v", destPath, a)
    }
  })
}

func TestMostRecentlyModifiedOverride(t *testing.T){
  srcPath, _ := createTestFiles(t, testFiles)
  defer os.RemoveAll(srcPath)

  destPath, _ := createTestFiles(t, testFiles2)
  defer os.RemoveAll(destPath)

  flagForcePath = 1
  a, _, _ := mostRecentlyModified("dir1/dir2/file3", srcPath, destPath)
  if a != srcPath {
    t.Errorf("Expected %v, got %v", srcPath, a)
  }

  flagForcePath = 2
  b, _, _ := mostRecentlyModified("file1", srcPath, destPath)
  if b != destPath {
    t.Errorf("Expected %v, got %v", destPath, b)
  }

  flagForcePath = 0
}

func TestRenameFolder(t *testing.T) {
  testFiles := []*TestFile{
    {"dir1/file1", "abcde", ""},
    {"dir2/file2", "a", ""},
    {"dir3/file3", "", ""},
    {"dir4/file4", "", ""},
    {"dir6/file6", "", ""},
  }

  dir, _ := createTestFiles(t, testFiles)
  defer os.RemoveAll(dir)

  tests := [][]string{
    {"dir2", "dir1", "dir1 (1)"},
    {"dir3", "dir1", "dir1 (2)"},
    {"dir4", "dir5", "dir5"},
    {"dir6", "path2/dir5", "path2/dir5"},
  }

  for i := 0; i < len(tests); i++ {
    r, err := RenameFolder(filepath.Join(dir, tests[i][0]), filepath.Join(dir, tests[i][1]))
    if err != nil {
      t.Errorf("Unexpected error %v", err.Error())
    }

    exp := filepath.Join(dir, tests[i][2])
    if r != exp {
      t.Errorf("Expected %v, got %v", exp, r)
    }
  }

  // test dest parent folder os.MkdirAll error
  _, err := RenameFolder("/notfound", "/not/found")
  if err == nil {
    t.Errorf("Expected error from os.MkdirAll")
  }
}
