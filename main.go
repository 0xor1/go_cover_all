package main

import(
	"os"
	"os/exec"
	"io/ioutil"
	"path/filepath"
	"strings"
	"bytes"
	"bufio"
	"log"
)

var(
	coverageFilePaths = []string{}
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalln("Couldn't get working directory", err)
	}

	runGoTestInThisAndAllSubDirectories(wd)

	//aggregate all the coverage results into one file in the root directory
	modeLineLength := -1
	aggregateCoverData := []byte{}
	for _, coverageFilePath := range coverageFilePaths {

		data, err := ioutil.ReadFile(coverageFilePath)

		if err != nil {
			log.Println("error reading file", coverageFilePath, err)
		}

		if modeLineLength == -1 {
			reader := bytes.NewReader(data)
			bufReader := bufio.NewReader(reader)
			modeLineBytes, err := bufReader.ReadBytes('\n')
			if err != nil {
				log.Println("error reading mode line", coverageFilePath, err)
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
		log.Println("Error writing aggregate coverprofile file", aggregateCoverprofilePath, err)
	} else {
		coverageFilePaths = append(coverageFilePaths, aggregateCoverprofilePath)
		log.Println("generating html report in", wd)
		cmd := exec.Command("go", "tool", "cover", "-html=aggregateCoverprofile.out", "-o=index.html")
		cmd.Dir = wd
		output := bytes.NewBuffer([]byte{})
		cmd.Stdout = output
		err := cmd.Run()
		log.Println(output.String())
		if err != nil {
			log.Println("error generating html report", err)
		}
	}

	//delete all the coverage files including
	for _, coverageFilePath := range coverageFilePaths {
		if err := os.Remove(coverageFilePath); err != nil {
			log.Println("Error deleting file", coverageFilePath, err)
		}
	}
}

func runGoTestInThisAndAllSubDirectories(path string){
	fileInfos, err := ioutil.ReadDir(path)
	if err != nil {
		log.Println("Couldn't read directory", err)
	}

	testsNotYetRunHere := true

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() && fileInfo.Name() != "vendor" && !strings.HasPrefix(fileInfo.Name(), "_") && fileInfo.Name() != ".git" && fileInfo.Name() != ".idea" {
			runGoTestInThisAndAllSubDirectories(filepath.Join(path, fileInfo.Name()))
		} else	if testsNotYetRunHere && strings.HasSuffix(fileInfo.Name(), "_test.go") {
			log.Println("running tests in", path)
			cmd := exec.Command("go", "test", "-coverprofile=coverage.out")
			cmd.Dir = path
			output := bytes.NewBuffer([]byte{})
			cmd.Stdout = output
			err := cmd.Run()
			log.Println(output.String())
			if err != nil {
				log.Println("error running tests", err)
			} else {
				coverageFilePaths = append(coverageFilePaths, filepath.Join(path, "coverage.out"))
			}
			testsNotYetRunHere = false
		}
	}
}