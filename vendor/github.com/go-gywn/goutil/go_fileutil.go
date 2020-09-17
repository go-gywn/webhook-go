package goutil

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
)

// FileUtil FileUtil
type FileUtil struct {
	absPath string
}

// GetFileUtil GetFileUtil
func GetFileUtil() FileUtil {
	return FileUtil{absPath: ""}
}

// GetABSPath GetABSPath
func (o *FileUtil) GetABSPath() string {
	if o.absPath == "" {
		r, _ := filepath.Abs(filepath.Dir(os.Args[0]))
		o.absPath = r
	}
	return o.absPath
}

// GetFilePath GetFilePath
func (o *FileUtil) GetFilePath(path string) string {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		path = fmt.Sprintf("%s/%s", o.GetABSPath(), path)
	}
	return path
}

// ReadFile read file content
func (o *FileUtil) ReadFile(path string) (r string) {
	var b []byte
	var err error
	b, err = ioutil.ReadFile(o.GetFilePath(path))
	if err != nil {
		return
	}
	r = string(b)
	return
}

// GetTemplate GetTemplate
func (o *FileUtil) GetTemplate(name string, templateString string) (t *template.Template, err error) {
	t, err = template.New(name).Parse(templateString)
	return
}
