package sfd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"github.com/itchyny/gojq"
)

const Query = "[select(.[].files != null) | .[].files | select(. != null) | {id: .[].id, url: .[].url_private_download}]"
const DirPrefix = "SlackAttachedFiles"

var filenameWithoutExtRegexp = regexp.MustCompile(`^\/?(?:.+\/)*(?P<date>\d{4}-\d{2}-\d{2})\.json$`)

type ResJson struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

func (r *ResJson) filename() (string, error) {
	u, err := url.Parse(r.Url)
	if err != nil {
		return "", fmt.Errorf("URL parse error: %w", err)
	}
	return fmt.Sprintf("%s_%s", r.Id, filepath.Base(u.Path)), nil
}

func (r *ResJson) getAttachedFile(saveDirPath string) error {
	if r.Url == "" {
		return nil
	}
	filename, err := r.filename()
	if err != nil {
		return fmt.Errorf("Create tempfile error: %w", err)
	}
	savePath := filepath.Join(saveDirPath, filename)
	if _, err := os.Stat(savePath); err == nil {
		fmt.Printf("Skip because the file already exists: %s\n", savePath)
		return nil
	}

	file, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("Create tempfile error: %w", err)
	}
	defer file.Close()

	res, err := http.Get(r.Url)
	if err != nil {
		return fmt.Errorf("Download exefile error: %w", err)
	}
	defer res.Body.Close()

	_, err = io.Copy(file, res.Body)
	if err != nil {
		return fmt.Errorf("Copy file error: %w", err)
	}
	fmt.Printf("Attachments have been saved: %s\n", savePath)
	return nil
}

func Run(path string) {
	logfilePaths, err := correctLogs(path)
	if err != nil {
		panic(err)
	}

	index := filenameWithoutExtRegexp.SubexpIndex("date")
	for _, logfilePath := range logfilePaths {
		b, err := os.ReadFile(logfilePath)
		if err != nil {
			panic(err)
		}
		fmt.Println("Start: ", logfilePath)
		correctJson, err := queryWithJQ(Query, b)
		if len(correctJson) < 1 {
			continue
		}

		matches := filenameWithoutExtRegexp.FindStringSubmatch(logfilePath)
		subDirName := matches[index]
		saveDir := filepath.Join(path, DirPrefix, subDirName)
		err = os.MkdirAll(saveDir, os.ModePerm)
		if err != nil {
			panic(err)
		}

		for _, j := range correctJson {
			j.getAttachedFile(saveDir)
		}
	}
}

func correctLogs(path string) ([]string, error) {
	var paths []string
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("Walk filepath error: %w", err)
		}
		if info.IsDir() {
			return nil
		}
		match := filenameWithoutExtRegexp.MatchString(info.Name())
		if !match {
			return nil
		}
		fmt.Println("path: ", p)
		paths = append(paths, p)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return paths, nil
}

func appendLogs(paths []string) (*bytes.Buffer, error) {
	var finalContent bytes.Buffer
	for _, path := range paths {
		content, err := ioutil.ReadFile(path)

		if err != nil {
			return nil, fmt.Errorf("Read file error: %w", err)
		}
		n, err := finalContent.Write(content)
		if err != nil {
			return nil, fmt.Errorf("Write buffer error: %w", err)
		}
		fmt.Printf("Success read file: %s: %d\n", path, n)
	}

	return &finalContent, nil
}

func queryWithJQ(query string, b []byte) ([]*ResJson, error) {
	var correctedJson []*ResJson
	q, err := gojq.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("Parse gojq query error: %w", err)
	}

	var tempJson interface{}
	if err := json.Unmarshal(b, &tempJson); err != nil {
		log.Fatalln(fmt.Errorf("Unmarshal json error: %w", err))
	}

	iter := q.Run(tempJson)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, fmt.Errorf("Iterate gojq error: %w", err)
		}

		casted := v.([]interface{})
		if len(casted) < 1 {
			continue
		}

		for _, e := range casted {
			var resJson ResJson

			b, err := json.Marshal(e.(map[string]interface{}))
			if err != nil {
				return nil, fmt.Errorf("JSON marshal error: %w", err)
			}

			err = json.Unmarshal(b, &resJson)

			if err != nil {
				return nil, fmt.Errorf("JSON unmarshal error: %w", err)
			}
			correctedJson = append(correctedJson, &resJson)
		}
	}

	return correctedJson, nil
}
