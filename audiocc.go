package main

import (
  "os"
  "fmt"
  "log"
  "image"
  "regexp"
  "strings"
  "runtime"
  "io/ioutil"
  "path/filepath"

  "github.com/JamTools/goff/ffmpeg"
  "github.com/JamTools/goff/ffprobe"
)

type audiocc struct {
  DirEntry string
  Image string
  Ffmpeg ffmpeg.Ffmpeger
  Ffprobe ffprobe.Ffprober
  Files []string
  Workers int
  Workdir string
}

func main() {
  args, cont := processFlags()
  if !cont {
    os.Exit(0)
  }

  ffm, err := ffmpeg.New()
  if err != nil {
    log.Fatal(err)
  }

  ffp, err := ffprobe.New()
  if err != nil {
    log.Fatal(err)
  }

  // process files using multiple cores
  var workers = runtime.NumCPU()
  //workers = 1 // DEBUG in single-threaded mode

  a := &audiocc{ Ffmpeg: ffm, Ffprobe: ffp, Workers: workers,
    DirEntry: filepath.Clean(args[0]) }

  err = a.process()
  if err != nil {
    log.Fatal(err)
  }
}

// true if --collection & artist path contains " - "
func skipArtistOnCollection(p string) bool {
  if flags.Collection {
    pa := strings.Split(p, sep)
    // first folder is artist
    if strings.Index(pa[0], " - ") != -1 {
      return true
    }
  }
  return false
}

// skip if album folder name contains year (--fast)
func skipFast(p string) bool {
  if flags.Fast {
    pa := strings.Split(p, sep)
    // last folder is album
    t := &info{}
    t.fromAlbum(pa[len(pa)-1])
    if len(t.Year) > 0 && len(t.Album) > 0 {
      return true
    }
  }
  return false
}

// process album art once per folder of files
func (a *audiocc) processArtwork(file string) error {
  art := &artwork{ Ffmpeg: a.Ffmpeg, Ffprobe: a.Ffprobe,
    ImgDecode: image.DecodeConfig,
    PathInfo: getPathInfo(a.DirEntry, file) }

  if flags.Write {
    var err error

    // probe to determine if has embedded artwork
    _, err = a.Ffprobe.GetData(art.PathInfo.Fullpath)
    if err != nil {
      return err
    }

    a.Image, err = art.process()
    if err != nil {
      return err
    }
  }

  return nil
}

func (a *audiocc) process() error {
  if !flags.Write {
    fmt.Printf("\n* To write changes to disk, please provide flag: --write\n")
  }

  // ensure path is is valid directory
  _, err := checkDir(a.DirEntry)
  if err != nil {
    return err
  }

  // obtain audio file list
  a.Files = filesByExtension(a.DirEntry, audioExts)

  // group files by parent directory
  err = bundleFiles(a.DirEntry, a.Files, func(indexes []int) error {
    pi := getPathInfo(a.DirEntry, a.Files[indexes[0]])

    fmt.Printf("\nProcessing: %v ...\n", pi.Fulldir)

    if skipArtistOnCollection(pi.Dir) || skipFast(pi.Dir) {
      return nil
    }

    // process artwork once per folder
    // TODO: if parent & child folders contain no audio files and are not the root dir,
    // check them for potential album artwork
    err = a.processArtwork(a.Files[indexes[0]])
    if err != nil {
      return err
    }

    // create new random workdir within current path
    a.Workdir, err = ioutil.TempDir(pi.Fulldir, "")
    if err != nil {
      return err
    }

    // TODO: threaded error handling
    path := a.processThreaded(indexes)
    dir := onlyDir(path)

    // remove workDir
    os.RemoveAll(a.Workdir)

    // if not same dir, rename directory to target dir
    if pi.Fulldir != dir {
      // TODO: only rename to (1) when duplicate track numbers exist
      // otherwise merge into existing directory
      _, err = renameFolder(pi.Fulldir, dir)
      return err
    }

    return nil
  })

  if err != nil {
    return err
  }

  fmt.Printf("\naudiocc finished.\n")
  return nil
}

func (a *audiocc) processThreaded(indexes []int) string {
  jobs := make(chan int)
  done := make(chan string, a.Workers)

  // iterate through files sending them to worker processes
  go func() {
    for x := range indexes {
      jobs <- indexes[x]
    }
    close(jobs)
  }()

  // start worker processes
  for i := 0; i < a.Workers; i++ {
    go func() {
      var dir string

      for job := range jobs {
        var err error
        dir, err = a.processIndex(job)
        if err != nil {
          fmt.Printf("\nError: %s\n", err.Error())
        }
      }

      // when jobs channel is closed
      done <- dir
    }()
  }

  // wait for all workers to finish
  var saveDir string
  for i := 0; i < a.Workers; i++ {
    saveDir = <-done
  }

  return saveDir
}

func (a *audiocc) processIndex(index int) (string, error) {
  pi := getPathInfo(a.DirEntry, a.Files[index])

  // info from path & filename
  mainInfo := &info{}
  mainInfo.fromFile(pi.File)
  mainInfo.fromPath(pi.Dir)

  // info from embedded tags within audio file
  d, err := a.Ffprobe.GetData(pi.Fullpath)
  if err != nil {
    return pi.Fullpath, err
  }

  i, match := mainInfo.matchProbeTags(d.Format.Tags)

  // skip if sources match (unless --force)
  if match && !flags.Force {
    return pi.Fullpath, nil
  }

  // override artist for consistency
  if len(flags.Artist) > 0 {
    // force artist which was specified
    i.Artist = flags.Artist
  } else if flags.Collection {
    // artist comes from parent folder name
    i.Artist = strings.Split(pi.Dir, sep)[0]
  }

  // build resulting path
  var path string
  if flags.Collection {
    // build from DirEntry; include artist then year
    path = filepath.Join(a.DirEntry, i.Artist, i.Year)
  } else {
    // remove current dir from fullpath
    path = strings.TrimSuffix(pi.Fulldir, pi.Dir)
  }

  // append directory generated from info
  path = filepath.Join(path, i.toAlbum())

  // print changes to be made
  p := fmt.Sprintf("%v\n", pi.Fullpath)
  if !match {
    p += fmt.Sprintf("  * update tags: %#v\n", i)
  }

  // convert audio (if necessary) & update tags
  if pi.Ext != ".flac" || regexp.MustCompile(` - FLAC$`).FindString(pi.Dir) == "" {
    // convert to mp3 except flac files with " - FLAC" in folder name
    f, err := a.processMp3(pi, i)
    if err != nil {
      return pi.Fullpath, err
    }

    // add filename to returning path
    _, file := filepath.Split(f)
    path = filepath.Join(path, file)

    if pi.Fullpath != path {
      p += fmt.Sprintf("  * rename to: %v\n", path)
    }
  } else {
    // TODO: use metaflac to edit flac metadata & embedd artwork
    p += fmt.Sprintf("\n*** Flac processing with 'metaflac' not yet implemented.\n")
  }

  // print to console all at once
  fmt.Printf(p)

  return path, nil
}

func (a *audiocc) processMp3(pi *pathInfo, i *info) (string, error) {
  // if already mp3, copy stream; do not convert
  quality := flags.Bitrate
  if pi.Ext == ".mp3" {
    quality = "copy"
  }

  // TODO: specify lower bitrate if source file is of low bitrate

  // build metadata from tag info
  ffmeta := ffmpeg.Metadata{ Artist: i.Artist, Album: i.toAlbum(),
    Disc: i.Disc, Track: i.Track, Title: i.Title, Artwork: a.Image }

  // save new file to Workdir subdir within current path
  newFile := filepath.Join(a.Workdir, i.toFile() + ".mp3")

  // process or convert to mp3
  c := &ffmpeg.Mp3Config{ pi.Fullpath, quality, newFile, ffmeta, flags.Fix }
  _, err := a.Ffmpeg.ToMp3(c)
  if err != nil {
    return newFile, err
  }

  // ensure output file was written
  fi, err := os.Stat(newFile)
  if err != nil {
    return newFile, err
  }

  // if flagsWrite & resulting file has size
  // TODO: ensure resulting file is good by reading & comparing metadata
  if fi.Size() > 0 {
    file := filepath.Join(pi.Fulldir, i.toFile() + ".mp3")

    if flags.Write {
      // delete original
      err = os.Remove(pi.Fullpath)
      if err != nil {
        return file, err
      }

      // move new to original directory
      err = os.Rename(newFile, file)
      if err != nil {
        return file, err
      }
    }

    return file, nil
  }

  return newFile, fmt.Errorf("File didn't have size")
}
