package iomonkey

import (
	"github.com/bdotdub/exiftool"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"math/rand"
	"fmt"
	"sync"
	"sort"
)

const (
	PHOTO_PREFIX		= "Photos"
	VIDEO_PREFIX		= "Videos"
	DATE_LAYOUT_FILEPATH = "2006/Jan"
	DATE_LAYOUT_EXIF     = "2006:01:02 15:04:05"
)

type FileMapping struct {
	LocalPath  string
	RemotePath string
}

type FileScanner struct {
	inProgress sync.RWMutex
	root string
	files map[string]*FileMapping
}

func NewFileScanner(root string) *FileScanner {
	fs := &FileScanner{
		root: root,
		files: make(map[string]*FileMapping),
	}
	fs.inProgress.Lock()
	go func(){
		defer fs.inProgress.Unlock()
		filepath.Walk(root, fs.walkFn)
	}()
	return fs
}



func (fs *FileScanner)Files() (<-chan*FileMapping,int) {
	fs.inProgress.RLock()
	keys := make([]string, 0, len(fs.files))
	for key := range fs.files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Println(keys)

	ch := make(chan*FileMapping)
	go func(){
		defer fs.inProgress.RUnlock()
		for _,key := range keys {
			ch <- fs.files[key]
		}
	}()
	return ch,len(fs.files)
}

func (fs *FileScanner)walkFn(path string, info os.FileInfo, err error) error {
	log.Println(path)
	if info.IsDir() {
		return nil
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		fs.handleFile(path,PHOTO_PREFIX)
	case ".mov", ".mp4":
		fs.handleFile(path,VIDEO_PREFIX)
	default:
		log.Printf("Skipping file: '%s'\n", path)
	}

	return nil
}

func (fs *FileScanner) handleFile(path string,prefix string){
	exif, err := exiftool.DecodeFileAtPath(path)
	if err != nil {
		log.Println(err)
		return
	}

	time, err := time.Parse(DATE_LAYOUT_EXIF, exif.DateTimeOriginal)
	if err != nil {
		log.Printf("[WARN] Exif parsing: %s\n", err)
	}

	fpath := "MISSING_EXIF"
	if !time.IsZero() {
		fpath = time.Format(DATE_LAYOUT_FILEPATH)
	}
	remotePath := filepath.Join(prefix, fpath, filepath.Base(path))
	for {
		if _, ok := fs.files[remotePath]; !ok {
			fs.files[remotePath] = &FileMapping{
				LocalPath:  path,
				RemotePath: remotePath,
			}
			break
		}
		dir,file := filepath.Split(remotePath)
		ext := filepath.Ext(file)
		file = fmt.Sprintf("%s%d%s",file[:len(file)-len(ext)],rand.Intn(9),ext)
		remotePath = filepath.Join(dir,file)
	}

}
