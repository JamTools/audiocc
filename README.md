# audioc

Clean up audio collection setting meta tags & embedding artwork

## Usage

```
Usage: audioc [MODE] [OPTIONS] PATH

Positional Args:
  PATH           directory path

MODE (specify only one):
  --artist "ARTIST" --album "ALBUM"
    treat as specific album belonging to specific artist

  --artist "ARTIST"
    treat as specific artist

  --collection
    treat as collection of artists

OPTIONS:
  --bitrate "BITRATE"
    V0 (default)
      convert to variable 256kbps mp3
    320
      convert to constant 320kbps mp3

  --fix
    fixes incorrect track length, ie 1035:36:51

  --force
    processes all files, even if path info matches tag info

  --write
    write changes to disk

Debug:
  --version
    print program version, then exit

```

## Purpose

This program is designed to process a music collection, keeping specified FLAC
audio files while converting all other audio formats to MP3.

Source albums have a release year or performance date. This date is then used
in both the album tag and folder path.

An example of the resulting folder structure:

```
Grateful Dead/
    1977/
        1977.05.15 St. Louis Arena, St. Louis, MO/
        1977 Terrapin Station/
```

In the above example, `1977.05.15 St. Louis Arena, St. Louis, MO` represents a
live performance, while `1977 Terrapin Station` represents a studio album. Both
belong to the artist `Grateful Dead`, nested within an additional folder
representing the year `1977`.

## Dependencies

This tool depends on `ffmpeg` and `ffprobe` binaries installed or included
within same folder, which are used to process the audio files and artwork.

To download `ffmpeg`:
[https://ffmpeg.org/download.html](https://ffmpeg.org/download.html)

The `metaflac` binary needs to be installed or included to support album
artwork embedding within FLAC files. If `metaflac` is not found, FLAC artwork
embedding will be skipped, but the program will continue without error.

The `metaflac` binary is part of the `flac` package.

To download `flac`:
[https://xiph.org/flac/download.html](https://xiph.org/flac/download.html)

## Mode

### Album (--artist "Artist Name" --album "Album Name")

Files nested within specified PATH are considered to be part of a specified
album or live performance belonging to a specified artist.

### Artist (--artist "Artist Name")

Child directories of specified PATH are considered to be albums or live
performances belonging to the specified artist.

### Collection (--collection)

Child directories of specified PATH are considered to be artists. Child
directories of each artist are considered to be albums or live performances
belonging to that artist.

The artist folder name overrides the audio file embedded artist metadata.

To skip processing a child directory, include ` - ` in its name. Such as:
`Grateful Dead - UNORGANIZED`

## Options

### Bitrate (--bitrate V0 OR --bitrate 320)

Convert other audio formats to MP3 using `libmp3lame` encoding and either V0
(variable 256kbps) or 320 (constant 320kbps) bitrate.

To skip converting FLAC audio to MP3, include ` - FLAC` at the end of the album
folder name.

### Fix (--fix)

Fixes incorrect track length (ie, 1035:36:51) affecting certain variable MP3
encodes by removing all metadata, then adding minimal metadata back in a
separate process.

### Force (--force)

Processes each audio file regardless of whether or not the path and file info
matches its tag info.

### Write (--write)

By not including `--write`, the process will run in simulation, printing all
changes to the console for review.

Including `--write` will apply changes by writing to disk. This process cannot
be undone.

## Developing

### Install / Update Go on Linux

Download latest go binary from [golang.org/dl](https://golang.org/dl/). In this case, version `1.13.5`.

Remove any existing installation, run:

    if [ -d /usr/local/go ]; then sudo rm -r /usr/local/go; fi

Extract to `/usr/local`, run:

    sudo tar -C /usr/local -xzf go1.13.5.linux-amd64.tar.gz

Create go home dir if doesn't already exist, run:

    if [ ! -d $HOME/go ]; then mkdir $HOME/go; fi

Edit `~/.profile`, run:

    nano ~/.profile

Append the following, then save/exit:

    export PATH=$PATH:/usr/local/go/bin
    export GOPATH=$(go env GOPATH)
    export PATH=$PATH:$GOPATH/bin

Source updated profile, run:

    source ~/.profile

### Building

From within source path, to build the binary, run:

    go build -o audioc cmd/*.go

To install the binary, run:

    go build -o $GOPATH/bin/audioc cmd/*.go

To test by displaying usage, run:

    audioc --help

### Testing

From within source path, run:

    go test -cover -v ./...

### Contributing

Fork repo on Github.

From within source path, setup new remote, run:

    git remote add myfork git@github.com:$GITHUB-USERNAME/audioc.git

Create a new branch to use for development, run:

    git checkout -b new-branch

Make your changes, add, commit and push to your Github fork.

Back on Github, submit pull request.

## License

This code is available open source under the terms of the
[MIT License](http://opensource.org/licenses/MIT).
