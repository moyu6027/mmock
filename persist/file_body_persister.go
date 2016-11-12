package persist

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jmartin82/mmock/definition"
	"github.com/jmartin82/mmock/parse"
)

//FileBodyPersister persists body in file
type FileBodyPersister struct {
	PersistPath string
	Parser      parse.ResponseParser
}

//Persist the body of the response to fiel if needed
func (fbp FileBodyPersister) Persist(per *definition.Persist, req *definition.Request, res *definition.Response) {
	if per.Name == "" {
		return
	}

	per.Name = fbp.Parser.ReplaceVars(req, per.Name)

	pathToFile := path.Join(fbp.PersistPath, per.Name)
	fileDir := path.Dir(pathToFile)

	if per.Delete {
		os.Remove(pathToFile)
	} else {
		fileContent := []byte(res.Body)
		err := os.MkdirAll(fileDir, 0644)
		if checkForError(err, res) == nil {
			err = ioutil.WriteFile(pathToFile, fileContent, 0644)
			checkForError(err, res)
		}
	}
}

func (fbp FileBodyPersister) getFilePath(fileName string, req *definition.Request) (pathToFile string, fileDir string) {
	fileName = fbp.Parser.ReplaceVars(req, fileName)

	pathToFile = path.Join(fbp.PersistPath, fileName)
	fileDir = path.Dir(pathToFile)

	return pathToFile, fileDir
}

//LoadBody loads the response body from the persisted file
func (fbp FileBodyPersister) LoadBody(req *definition.Request, res *definition.Response) {
	pathToFile, _ := fbp.getFilePath(res.Persisted.Name, req)

	var err error
	pathToFile, err = filepath.Abs(pathToFile)
	if checkForError(err, res) != nil {
		return
	}
	if !strings.HasPrefix(pathToFile, fbp.PersistPath) {
		errorText := fmt.Sprintf("File path not under the persist path. FilePath: %s, PersistPath %s", pathToFile, fbp.PersistPath)
		checkForError(errors.New(errorText), res)
		return
	}

	_, err = os.Stat(pathToFile)

	// use notFound info
	if os.IsNotExist(err) {
		res.Body = "Not Found"
		if res.Persisted.NotFound.Body != "" {
			res.Body = fbp.Parser.ParseBody(req, res.Persisted.NotFound.Body, res.Persisted.NotFound.BodyAppend)
		}
		res.StatusCode = 404
		if res.Persisted.NotFound.StatusCode != 0 {
			res.StatusCode = res.Persisted.NotFound.StatusCode
		}
	} else {
		fileContent, err := ioutil.ReadFile(pathToFile)
		if checkForError(err, res) == nil {
			res.Body = fbp.Parser.ParseBody(req, string(fileContent), res.Persisted.BodyAppend)
		}
	}
}

func checkForError(err error, res *definition.Response) error {
	if err != nil {
		log.Print(err)
		res.Body = err.Error()
		res.StatusCode = 500
	}
	return err
}

//NewFileBodyPersister creates a new FileBodyPersister
func NewFileBodyPersister(persistPath string, parser parse.ResponseParser) *FileBodyPersister {
	result := FileBodyPersister{PersistPath: persistPath, Parser: parser}

	err := os.MkdirAll(result.PersistPath, 0644)
	if err != nil {
		panic(err)
	}

	return &result
}
