package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

const (
	kimaiTimesheetsPath = "/timesheets"
	configFileName = "notimefy.json"
)

var (
	config	Config
)

type Config struct {
	KimaiUrl	string
	KimaiUsername	string
	KimaiPassword	string
}

type KimaiRecord struct {
	Duration	int
}

func readConfig(configPath string) {
	if len(configPath) == 0 {
		configDir := getHomePath() // TODO: first try the root of the repo looking for the file
		err := os.MkdirAll(configDir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
		configPath = filepath.Join(configDir, configFileName)
	}

	configFile, err := os.Open(configPath)
	if err != nil {
		log.Fatal(err)
	}
	defer configFile.Close()

	configBytes, err := io.ReadAll(configFile)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		log.Fatal(err)
	}

	if config.KimaiUrl == "" {
		log.Fatalln("No Kimai URL specified in the config file")
	}
	if config.KimaiUsername == "" {
		log.Fatalln("No Kimai username specified in the config file")
	}
	if config.KimaiPassword == "" {
		log.Fatalln("No Kimai password specified in the config file")
	}
}

func getHomePath() string {
	var homePath string
	if runtime.GOOS == "windows" {
		homePath = "HOMEPATH"
	} else {
		homePath = "HOME"
	}

	return filepath.Join(os.Getenv(homePath), ".config")
}

func getCurrentMonthDayOneDate() string {
	monthAndYear := time.Now().Format("2006-01")
	return monthAndYear + "-01T00:00:00"
}

func getNow() string {
	return time.Now().Format("2006-01-02T15:04:05")
}

func fetchKimaiResource(url string, method string, body io.Reader) []byte {
	client := &http.Client{}
	httpReq, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Fatal(err)
	}

	httpReq.Header.Set("X-AUTH-USER", config.KimaiUsername)
	httpReq.Header.Set("X-AUTH-TOKEN", config.KimaiPassword)

	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json") 
	}

	resp, err := client.Do(httpReq)
	if err != nil {	
		log.Fatal(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return respBody
}

func fetchKimaiMonthRecords() []KimaiRecord {
	var monthRecords []KimaiRecord
	timeArgs := "?begin=" + getCurrentMonthDayOneDate() + "&end=" + getNow()

	i := 1
	for {
		pageArg := "&page=" + strconv.Itoa(i)
		args := timeArgs + pageArg
		url := config.KimaiUrl + kimaiTimesheetsPath + args
		method := "GET"

		respBody := fetchKimaiResource(url, method, nil)

		var pageMonthRecords []KimaiRecord
		err := json.Unmarshal(respBody, &pageMonthRecords)
		if err != nil {
			break
		}
		monthRecords = append(monthRecords, pageMonthRecords...)
		i++
	}

	return monthRecords
}

func monthDurationTotal() int {
	monthRecords := fetchKimaiMonthRecords()
	var durationTotal int 
	for i := 0; i < len(monthRecords); i++ {
		durationTotal += monthRecords[i].Duration
	}

	return durationTotal
}

func hoursFromMinutesDuration(minutesDuration int) int {
	hours := 0
	if minutesDuration > 59 {
		hours = minutesDuration / 60
	}

	return hours
}

func main() {
	readConfig("")
	fmt.Println(hoursFromMinutesDuration(monthDurationTotal()))
}
