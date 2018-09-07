package main

import (
  "io"
  "os"
  "reflect"
  "strconv"
  "testing"
  "io/ioutil"
  "path/filepath"

  "github.com/JamTools/goff/fsutil"
)

func tmpFile(t *testing.T, input string, f func(in *os.File)) {
  in, err := ioutil.TempFile("", "")
  if err != nil {
    t.Fatal(err)
  }
  defer os.Remove(in.Name())
  defer in.Close()

  _, err = io.WriteString(in, input)
  if err != nil {
    t.Fatal(err)
  }

  _, _ = in.Seek(0, os.SEEK_SET)

  f(in)
}

func TestPathInfo(t *testing.T) {
  tests := []struct {
    artist, base, path string
    pi *pathInfo
  }{
    { base: "dir1", path: "dir2/dir3/file1.ext",
      pi: &pathInfo{ Fullpath: "dir1/dir2/dir3/file1.ext",
        Fulldir: "dir1/dir2/dir3", Dir: "dir2/dir3", File: "file1", Ext: ".ext" },
    },{
      base: "dir3/dir4", path: "file2.ext",
      pi: &pathInfo{ Fullpath: "dir3/dir4/file2.ext",
        Fulldir: "dir3/dir4", Dir: "dir4", File: "file2", Ext: ".ext" },
    },{
      base: "/dir3/dir4/", path: "file2.ext",
      pi: &pathInfo{ Fullpath: "/dir3/dir4/file2.ext",
        Fulldir: "/dir3/dir4", Dir: "dir4", File: "file2", Ext: ".ext" },
    },{
      artist: "Artist", base: "/dir3/dir4/", path: "dir5/file2.ext",
      pi: &pathInfo{ Fullpath: "/dir3/dir4/dir5/file2.ext",
        Fulldir: "/dir3/dir4/dir5", Dir: "dir4/dir5", File: "file2", Ext: ".ext" },
    },
  }

  for x := range tests {
    flags.Artist = tests[x].artist
    defer func() { flags.Artist = "" }()

    pi := getPathInfo(tests[x].base, tests[x].path)
    if !reflect.DeepEqual(pi, tests[x].pi) {
      t.Errorf("Expected %v, got %v", tests[x].pi, pi)
    }
  }
}

func TestCheckDirInvalid(t *testing.T) {
  // not exist
  _, err := checkDir("audiocc-path-def-dne")
  if err == nil {
    t.Errorf("Expected error, got nil")
  }

  // not directory
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

func TestOnlyDir(t *testing.T) {
  path := filepath.Join("one", "two", "three.jpg")
  r := onlyDir(path)
  if r != "one/two" {
    t.Errorf("Expected %v, got %v", "one/two", r)
  }
}

func TestBundleFiles(t *testing.T) {
  testFiles := []string{
    "artist1/file1",
    "artist1/file2",
    "artist1/file3",
    "artist2/file1",
    "artist2/file2",
    "artist3/file1",
    "artist4/file1",
  }

  bundles := []string{
    "012",
    "34",
    "5",
    "6",
  }

  results := make([]string, 0)
  _ = bundleFiles("/test", testFiles, func(bundle []int) error {
    var r string
    for i := range bundle {
      r += strconv.Itoa(bundle[i])
    }

    results = append(results, r)

    // TODO: test returning error
    return nil
  })

  err := false
  for x := range bundles {
    if x > len(results)-1 || bundles[x] != results[x] {
      err = true
      break
    }
  }

  if err {
    t.Errorf("Expected %v, got %v", bundles, results)
  }
}

// TODO update this
func TestSafeFilename(t *testing.T) {
  tests := [][]string{
    { "", "" },
  }

  for i := range tests {
    r := safeFilename(tests[i][0])
    if r != tests[i][1] {
      t.Errorf("Expected %v, got %v", tests[i][1], r)
    }
  }
}

func TestRenameFolder(t *testing.T) {
  testFiles := []*fsutil.TestFile{
    {"dir1/file1", "abcde"},
    {"dir2/file2", "a"},
    {"dir3/file3", ""},
    {"dir4/file4", ""},
    {"dir6/file6", ""},
  }

  dir := fsutil.CreateTestFiles(t, testFiles)
  defer os.RemoveAll(dir)

  tests := [][]string{
    {"dir2", "dir1", "dir1 (1)"},
    {"dir3", "dir1", "dir1 (2)"},
    {"dir4", "dir5", "dir5"},
    {"dir6", "path2/dir5", "path2/dir5"},
  }

  for i := 0; i < len(tests); i++ {
    r, err := renameFolder(filepath.Join(dir, tests[i][0]), filepath.Join(dir, tests[i][1]))
    if err != nil {
      t.Errorf("Unexpected error %v", err.Error())
    }

    exp := filepath.Join(dir, tests[i][2])
    if r != exp {
      t.Errorf("Expected %v, got %v", exp, r)
    }
  }
}
