package main

import (
	"cacheDatabase"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
)

type File struct {
	cacheDatabase.FileTemplate

	Size         int64
	DateModified int64
	Mode         int32
}

// Scan path result into database
func FileScan(scanPath string) error {

	scanPathAbs, err := filepath.Abs(scanPath)
	if err != nil {
		return err
	}
	err = filepath.Walk(scanPathAbs, func(path string, f os.FileInfo, err error) error {
		p := "/" + strings.Replace(path, "\\", "/", -1)
		fmt.Println(p, err)

		fileInterfaceDatabase, err := cacheDatabase.CreatePath(&File{}, p)
		var fileDatabase = fileInterfaceDatabase.(*File)

		fileDatabase.DateModified = f.ModTime().Unix()
		fileDatabase.Name = f.Name()
		fileDatabase.Size = f.Size()
		fileDatabase.Mode = int32(f.Mode())
		fileDatabase.Path = p
		fmt.Println("path: ", p)
		_, err = cacheDatabase.SavePath(fileInterfaceDatabase)

		if err != nil {
			glog.Error(path, ":", err)
		}
		return nil

	})
	cacheDatabase.CommitData()
	return err
}
