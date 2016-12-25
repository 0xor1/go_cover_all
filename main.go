package main

import(
	"github.com/uber-go/zap"
	"os"
	"os/exec"
	"io/ioutil"
	"path/filepath"
	"strings"
	"bytes"
	"bufio"
)

var(
	coverageFilePaths = []string{}
	log = zap.New(zap.NewTextEncoder())
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("Couldn't get working directory", zap.Error(err))
	}

	runGoTestInThisAndAllSubDirectories(wd)

	//aggregate all the coverage results into one file in the root directory
	modeLineLength := -1
	aggregateCoverData := []byte{}
	for _, coverageFilePath := range coverageFilePaths {

		data, err := ioutil.ReadFile(coverageFilePath)

		if err != nil {
			log.Error("error reading file", zap.String("file", coverageFilePath), zap.Error(err))
		}

		if modeLineLength == -1 {
			reader := bytes.NewReader(data)
			bufReader := bufio.NewReader(reader)
			modeLineBytes, err := bufReader.ReadBytes('\n')
			if err != nil {
				log.Error("error reading mode line", zap.String("file", coverageFilePath), zap.Error(err))
			} else {
				modeLineLength = len(modeLineBytes)
			}
		}

		if len(aggregateCoverData) == 0 {
			aggregateCoverData = append(aggregateCoverData, data...)
		} else {
			aggregateCoverData = append(aggregateCoverData, data[modeLineLength:]...)
		}

	}

	//generate the index.html cover results file, by writing out the aggregated data and running the go cover tool
	aggregateCoverprofilePath := filepath.Join(wd, "aggregateCoverprofile.out")
	if err := ioutil.WriteFile(aggregateCoverprofilePath, aggregateCoverData, os.ModePerm); err != nil {
		log.Error("Error writing aggregate coverprofile file", zap.String("file", aggregateCoverprofilePath), zap.Error(err))
	} else {
		coverageFilePaths = append(coverageFilePaths, aggregateCoverprofilePath)
		log.Info("generating html report in", zap.String("directory", wd))
		cmd := exec.Command("go", "tool", "cover", "-html=aggregateCoverprofile.out", "-o=index.html")
		cmd.Dir = wd
		output := bytes.NewBuffer([]byte{})
		cmd.Stdout = output
		err := cmd.Run()
		log.Info("", zap.String("output", output.String()))
		if err != nil {
			log.Error("error generating html report", zap.Error(err))
		}
	}

	//delete all the coverage files including
	for _, coverageFilePath := range coverageFilePaths {
		if err := os.Remove(coverageFilePath); err != nil {
			log.Error("Error deleting file", zap.String("file", coverageFilePath), zap.Error(err))
		}
	}
}

func runGoTestInThisAndAllSubDirectories(path string){
	log.Info("processing", zap.String("directory", path))
	fileInfos, err := ioutil.ReadDir(path)
	if err != nil {
		log.Error("Couldn't read directory", zap.Error(err))
	}

	testsNotYetRunHere := true

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() && fileInfo.Name() != "vendor" && !strings.HasPrefix(fileInfo.Name(), "_") && fileInfo.Name() != ".git" && fileInfo.Name() != ".idea" {
			runGoTestInThisAndAllSubDirectories(filepath.Join(path, fileInfo.Name()))
		} else	if testsNotYetRunHere && strings.HasSuffix(fileInfo.Name(), "_test.go") {
			log.Info("running tests in", zap.String("directory", path))
			cmd := exec.Command("go", "test", "-coverprofile=coverage.out")
			cmd.Dir = path
			output := bytes.NewBuffer([]byte{})
			cmd.Stdout = output
			err := cmd.Run()
			log.Info("", zap.String("output", output.String()))
			if err != nil {
				log.Error("error running tests", zap.Error(err))
			}
			coverageFilePaths = append(coverageFilePaths, filepath.Join(path, "coverage.out"))
			testsNotYetRunHere = false
		}
	}
}