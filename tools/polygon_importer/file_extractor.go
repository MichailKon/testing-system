package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func moveSingleFile(src *zip.File, dst string) error {
	w, err := os.OpenFile(
		dst,
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
		0664,
	)
	if err != nil {
		return err
	}
	defer w.Close()

	r, err := src.Open()
	if err != nil {
		return err
	}
	defer r.Close()

	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}

	return nil
}

func extractAllFiles(prefix string, dst string) error {
	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}

		if strings.HasPrefix(file.Name, prefix) {
			dstPath, err := filepath.Rel(prefix, file.Name)
			if err != nil {
				panic(err)
			}

			dstPath = filepath.Join(probTmpPath, dst, dstPath)
			err = os.MkdirAll(filepath.Dir(dstPath), 0775)
			if err != nil {
				return err
			}

			err = moveSingleFile(file, dstPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func moveFile(srcPath string, dstDir string, name string) error {
	dstDir = filepath.Join(probTmpPath, dstDir)
	err := os.MkdirAll(dstDir, 0775)
	if err != nil {
		return fmt.Errorf("can not create dir %s for file %s, error: %s", dstDir, srcPath, err.Error())
	}

	for _, file := range archive.File {
		if file.Name == srcPath {
			err = moveSingleFile(file, filepath.Join(dstDir, name))
			if err != nil {
				return fmt.Errorf("can not extract zip file %s to dir %s, error: %s", srcPath, dstDir, err.Error())
			}
			return nil
		}
	}
	return fmt.Errorf("file with path %s not found", srcPath)
}
