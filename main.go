package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/sqweek/dialog"
)

var audioContext = audio.NewContext(44100)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		dirpath, err := dialog.Directory().Title("Select a directory to cleanse").Browse()
		if err != nil {
			log.Fatal(errors.New("filepath required"))
		}
		args = append(args, dirpath)
	}

	var allFiles []string

	var recurse func(fullpath, localpath string)
	recurse = func(fullpath, localpath string) {
		fpath := path.Join(fullpath, localpath)
		files, err := ioutil.ReadDir(fpath)
		if err != nil {
			log.Println("couldn't read dir", err, fpath)
			return
		}
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".ogg") {
				allFiles = append(allFiles, path.Join(fpath, file.Name()))
			}
			if file.IsDir() {
				recurse(fpath, file.Name())
			}
		}

	}

	// Let's collect our population.
	recurse("", args[0])

	// Let's purge the non-believers.
	ok := dialog.Message("Potentially cleansing %d files", len(allFiles)).Title("Confirm Eradication.").YesNo()
	if !ok {
		return
	}
	var errs []error
	var removed []string
	for _, file := range allFiles {
		f, err := os.Open(file)
		if err != nil {
			fmt.Println("file couldnt be opened", err, file)
			errs = append(errs, fmt.Errorf("file '%s' couldn't be opened: %w", file, err))
			f.Close()
			continue
		}
		s, err := vorbis.DecodeWithSampleRate(audioContext.SampleRate(), f)
		if err != nil {
			fmt.Println("errored decoding", err, file)
			errs = append(errs, fmt.Errorf("file '%s' couldn't be decoded: %w", file, err))
			f.Close()
			continue
		}

		var b bytes.Buffer
		_, err = io.Copy(&b, s)
		if err != nil {
			fmt.Println("errored reading", err, file)
			errs = append(errs, fmt.Errorf("file '%s' couldn't be read: %w", file, err))
			f.Close()
			continue
		}

		f.Close()

		isEmpty := true
		for _, b := range b.Bytes() {
			if b != 0 {
				isEmpty = false
			}
		}
		if isEmpty {
			err := os.Remove(file)
			if err != nil {
				fmt.Println("errored removing", err, file)
				errs = append(errs, fmt.Errorf("file '%s' couldn't be removed: %w", file, err))
			} else {
				removed = append(removed, file)
			}
		}
	}

	if len(errs) > 0 {
		var errorString string
		for _, e := range errs {
			errorString = errorString + "\n" + fmt.Sprintf("%s", e)
		}
		dialog.Message("%s", errorString).Error()
	}
	var removedString string
	for _, s := range removed {
		removedString = removedString + "\n" + s
	}
	dialog.Message("%d out of %d files removed \n%s", len(removed), len(allFiles), removedString).Info()
}
