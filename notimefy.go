package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

const (
	kimaiTimesheetsPath = "/timesheets"
	configFileName      = ".config/notimefy.json" // TODO: only file
)

var (
	config Config
)

type Config struct {
	KimaiUrl       string
	KimaiUsername  string
	KimaiPassword  string
	SMTPUsername   string
	SMTPPassword   string
	SMTPHost       string
	SMTPPort       string
	RecipientEmail string
	HourThresholds []int
}

type KimaiRecord struct {
	Duration int
}

type PrevData struct {
	Month               string
	RemainingThresholds []int
}

func readConfig(configPath string) {
	if len(configPath) == 0 {
		configDir := getHomePath() // TODO: getConfigDir
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
	if config.SMTPUsername == "" {
		log.Fatalln("No SMTP username specified in the config file")
	}
	if config.SMTPPassword == "" {
		log.Fatalln("No SMTP password specified in the config file")
	}
	if config.SMTPHost == "" {
		log.Fatalln("No SMTP host specified in the config file")
	}
	if config.SMTPPort == "" {
		log.Fatalln("No SMTP port specified in the config file")
	}
	if config.RecipientEmail == "" {
		log.Fatalln("No recipient email specified in the config file")
	}
	if config.HourThresholds == nil {
		log.Fatalln("No hour thresholds specified in the config file")
	}
}

func getHomePath() string {
	var homePath string
	if runtime.GOOS == "windows" {
		homePath = "HOMEPATH"
	} else {
		homePath = "HOME"
	}

	return os.Getenv(homePath) // TODO: add .config
}

func getDataFilePath() string {
	parsedUrl, err := url.Parse(config.KimaiUrl)
	if err != nil {
		log.Fatal(err)
	}

	host := parsedUrl.Hostname()
	dataDir := filepath.Join(getHomePath(), ".local", "share", "notimefy")
	err = os.MkdirAll(dataDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	return filepath.Join(dataDir, host)
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

	return durationTotal / 60
}

func hoursFromMinutesDuration(minutesDuration int) int {
	hours := 0
	if minutesDuration > 59 {
		hours = minutesDuration / 60
	}

	return hours
}

func resetPrevData() {
	dataFilePath := getDataFilePath()
	os.Remove(dataFilePath)
}

func sendNotification(lastThresholdStr string, hoursStr string, monthAndYear string) {
	host := config.SMTPHost
	toStr := config.RecipientEmail
	to := []string{toStr}
	dateStr := monthAndYear + "-01"
	message := []byte("To: " + toStr + "\r\n" +
		"Subject: " + lastThresholdStr + " hours threshold surpassed since " + dateStr + "\r\n" +
		"\r\n" + "Surpassed " + lastThresholdStr + " hours (currently: " + hoursStr + " hours) " +
		"in " + monthAndYear + "\r\n")

	auth := smtp.PlainAuth("", config.SMTPUsername, config.SMTPPassword, host)
	err := smtp.SendMail(host+":"+config.SMTPPort, auth, config.SMTPUsername, to, message)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Email sent: " + string(message))
}

func notifyIfNecessary() {
	var prevData PrevData
	dataFilePath := getDataFilePath()
	prevDataBytes, err := os.ReadFile(dataFilePath)
	if err == nil {
		err = json.Unmarshal(prevDataBytes, &prevData)
	}

	currentMonth := time.Now().Format("2006-01")
	if err != nil || prevData.Month != currentMonth {
		prevData.Month = currentMonth
		prevData.RemainingThresholds = config.HourThresholds
	}

	hours := hoursFromMinutesDuration(monthDurationTotal())
	var remainingThresholds []int
	lastSurpassedThreshold := 0
	for i := 0; i < len(prevData.RemainingThresholds); i++ {
		remThreshold := prevData.RemainingThresholds[i]
		if hours >= remThreshold {
			lastSurpassedThreshold = remThreshold
		} else {
			remainingThresholds = append(remainingThresholds, remThreshold)
		}
	}

	if lastSurpassedThreshold > 0 && len(remainingThresholds) != len(prevData.RemainingThresholds) {
		prevData.RemainingThresholds = remainingThresholds
		lastThresholdStr := strconv.Itoa(lastSurpassedThreshold)
		hoursStr := strconv.Itoa(hours)
		sendNotification(lastThresholdStr, hoursStr, currentMonth)
	}

	prevDataBytes, err = json.Marshal(prevData)
	if err != nil {
		log.Fatal(err)
	}
	os.WriteFile(dataFilePath, prevDataBytes, 0666)
}

func main() { // TODO: simple reset option
	configPathPtr := flag.String("config", "", "Path to the configuration file")
	resetOpPtr := flag.Bool("reset-first", false, "Reset program state before running")
	flag.Parse()

	readConfig(*configPathPtr)
	if *resetOpPtr {
		resetPrevData()
	}
	notifyIfNecessary()
}
