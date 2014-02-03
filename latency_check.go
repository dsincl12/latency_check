package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var url string
var measurementStart = time.Now()
var totalNumberOfLatencyChecks = 0

type Report struct {
	URL                        string
	AverageLatency             string
	MeasurementDuration        string
	TotalNumberOfLatencyChecks string
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("usage: latency_check -t <interval in seconds> -u <url> [-w]")
		fmt.Println("-w enables a JSON web response on port 3000 which is a report of the latest information")
		return
	}

	intervalPtr := flag.Int("t", 10, "")
	webPtr := flag.Bool("w", false, "")
	flag.StringVar(&url, "u", "", "")

	flag.Parse()

	ticker := time.NewTicker(time.Duration(*intervalPtr) * time.Second)

	if *webPtr == true {
		http.HandleFunc("/", webHandler)
		go http.ListenAndServe(":3000", nil)
	}

	var latencies []time.Duration

	for {
		select {
		case <-ticker.C:
			latency := measureLatency(url)
			log.Println(url, "- Last latency:", latency)

			latencies = append(latencies, latency)
			latencies = compact(latencies) // only keep last 100

			average := calculateAverageLatency(latencies)
			log.Println("Average latency:", average)

			saveAverage(average)
			totalNumberOfLatencyChecks++
		}
	}
}

func webHandler(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open("latency.log")
	if err != nil {
		panic(err)
	}

	defer f.Close()

	reader := bufio.NewReader(f)
	line, _, err := reader.ReadLine()
	if err != nil {
		panic(err)
	}

	averageLatency := string(line)
	measurementDuration := time.Since(measurementStart)

	report := Report{url, averageLatency, measurementDuration.String(), strconv.Itoa(totalNumberOfLatencyChecks)}

	js, err := json.Marshal(report)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func measureLatency(url string) time.Duration {
	start := time.Now()

	_, err := http.Head(url)
	if err != nil {
		fmt.Println("Error:", url, err)
	}

	return time.Since(start)
}

func calculateAverageLatency(latencies []time.Duration) time.Duration {
	var sum int64
	for _, latency := range latencies {
		sum += int64(latency)
	}

	return time.Duration(sum / int64(len(latencies)))
}

func saveAverage(average time.Duration) {
	f, err := os.Create("latency.log")
	if err != nil {
		panic(err)
	}

	defer f.Close()

	f.WriteString(average.String())
}

func compact(latencies []time.Duration) []time.Duration {
	if len(latencies) == 100 {
		latencies = latencies[1:len(latencies)]
	}

	return latencies
}
