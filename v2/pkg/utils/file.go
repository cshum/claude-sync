package utils

import (
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
)

func GetLocalFiles(rootPath string) (map[string]string, error) {
	files := make(map[string]string)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(rootPath, path)
			if err != nil {
				return err
			}

			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			hash := md5.Sum(content)
			files[relPath] = hex.EncodeToString(hash[:])
		}

		return nil
	})

	return files, err
}
