package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pemistahl/lingua-go"
)

func generateFileList(dir string) ([]string, error) {
	var fileList []string
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			fileList = append(fileList, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Strings(fileList)
	return fileList, nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = srcFile.WriteTo(dstFile)
	if err != nil {
		return err
	}
	return nil
}

func copyDir(src, dst string) error {
	filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath := strings.TrimPrefix(path, src)
		dstPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			err := os.MkdirAll(dstPath, os.ModePerm)
			if err != nil {
				return err
			}
		} else {
			err := copyFile(path, dstPath)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return nil
}

func mapDetectedLanguage(detectedLang lingua.Language) string {
	switch detectedLang {
	case lingua.English:
		return "en"
	case lingua.Spanish:
		return "es"
	case lingua.Chinese:
		return "zh"
	case lingua.Korean:
		return "ko"
	case lingua.Japanese:
		return "ja"
	case lingua.German:
		return "de"
	case lingua.Russian:
		return "ru"
	case lingua.French:
		return "fr"
	case lingua.Dutch:
		return "nl"
	case lingua.Italian:
		return "it"
	case lingua.Indonesian:
		return "id"
	case lingua.Portuguese:
		return "pt"
	case lingua.Swedish:
		return "sv"
	default:
		return "en"
	}
}
