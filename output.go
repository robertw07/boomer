package boomer

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Output is primarily responsible for printing test results to different destinations
// such as consoles, files. You can write you own output and add to boomer.
// When running in standalone mode, the default output is ConsoleOutput, you can add more.
// When running in distribute mode, test results will be reported to master with or without
// an output.
// All the OnXXX function will be call in a separated goroutine, just in case some output will block.
// But it will wait for all outputs return to avoid data lost.
type Output interface {
	// OnStart will be call before the test starts.
	OnStart()

	// By default, each output receive stats data from runner every three seconds.
	// OnEvent is responsible for dealing with the data.
	OnEvent(data map[string]interface{})

	// OnStop will be called before the test ends.
	OnStop()
}

// ConsoleOutput is the default output for standalone mode.
type ConsoleOutput struct {
}

type OutputOptions struct {
	PercentTime int
}

var OutputOps = new(OutputOptions)

// NewConsoleOutput returns a ConsoleOutput.
func NewConsoleOutput() *ConsoleOutput {
	return &ConsoleOutput{}
}

func NewConsoleOutputWithOptions(outputOps *OutputOptions) *ConsoleOutput {
	OutputOps = outputOps
	if OutputOps.PercentTime <= 0 {
		OutputOps.PercentTime = 90
	}
	return &ConsoleOutput{}

}

func getMedianResponseTime(numRequests int64, responseTimes map[int64]int64) int64 {
	medianResponseTime := int64(0)
	if len(responseTimes) != 0 {
		pos := (numRequests - 1) / 2
		var sortedKeys []int64
		for k := range responseTimes {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Slice(sortedKeys, func(i, j int) bool {
			return sortedKeys[i] < sortedKeys[j]
		})
		for _, k := range sortedKeys {
			if pos < responseTimes[k] {
				medianResponseTime = k
				break
			}
			pos -= responseTimes[k]
		}
	}
	return medianResponseTime
}

func getAvgResponseTime(numRequests int64, totalResponseTime int64) (avgResponseTime float64) {
	avgResponseTime = float64(0)
	if numRequests != 0 {
		avgResponseTime = float64(totalResponseTime) / float64(numRequests)
	}
	return avgResponseTime
}

func getAvgContentLength(numRequests int64, totalContentLength int64) (avgContentLength int64) {
	avgContentLength = int64(0)
	if numRequests != 0 {
		avgContentLength = totalContentLength / numRequests
	}
	return avgContentLength
}

func getPercentResponseTime(numRequests int64, responseTimes map[int64]int64) int64 {
	medianResponseTime := int64(0)
	if len(responseTimes) != 0 {
		pos := (numRequests - 1) * (int64(OutputOps.PercentTime)) / 100
		var sortedKeys []int64
		for k := range responseTimes {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Slice(sortedKeys, func(i, j int) bool {
			return sortedKeys[i] < sortedKeys[j]
		})
		for _, k := range sortedKeys {
			if pos < responseTimes[k] {
				medianResponseTime = k
				break
			}
			pos -= responseTimes[k]
		}
	}
	return medianResponseTime
}

func getCurrentRps(numRequests int64, numReqsPerSecond map[int64]int64) (currentRps int64) {
	currentRps = int64(0)

	//fix accuracy issue by robert
	//var count int64
	//if len(numReqsPerSecond) > 1 {
	//	count = int64(len(numReqsPerSecond) - 1)
	//} else {
	//	count = int64(len(numReqsPerSecond))
	//}
	numReqsPerSecondLength := int64(len(numReqsPerSecond))
	if numReqsPerSecondLength != 0 {
		currentRps = numRequests / numReqsPerSecondLength
	}
	//currentRps = numRequests / int64(len(numReqsPerSecond))
	return currentRps
}

func getCurrentFailPerSec(numFailures int64, numFailPerSecond map[int64]int64) (currentFailPerSec int64) {
	currentFailPerSec = int64(0)
	numFailPerSecondLength := int64(len(numFailPerSecond))
	if numFailPerSecondLength != 0 {
		currentFailPerSec = numFailures / numFailPerSecondLength
	}
	return currentFailPerSec
}

func getTotalFailRatio(totalRequests, totalFailures int64) (failRatio float64) {
	if totalRequests == 0 {
		return 0
	}
	return float64(totalFailures) / float64(totalRequests)
}

// OnStart of ConsoleOutput has nothing to do.
func (o *ConsoleOutput) OnStart() {

}

// OnStop of ConsoleOutput has nothing to do.
func (o *ConsoleOutput) OnStop() {

}

// OnEvent will print to the console.
func (o *ConsoleOutput) OnEvent(data map[string]interface{}) {
	output, err := convertData(data)
	if err != nil {
		log.Println(fmt.Sprintf("convert data error: %v", err))
		return
	}

	output.Stats = sortOutput(output.Stats)

	buildAllStats(output)
	sortOutput(allStats.Stats)

	currentTime := time.Now()
	computerMonitor := GetCpuMem()
	println(fmt.Sprintf("Current Data: %s, Users: %d, Total RPS: %d, Total Fail Ratio: %.1f%%",
		currentTime.Format("2006/01/02 15:04:05"), output.UserCount, output.TotalRPS, output.TotalFailRatio*100))
	println(fmt.Sprintf("Summary data: %s, Users: %d, Total RPS: %d, Total Fail Ratio: %.1f%%",
		currentTime.Format("2006/01/02 15:04:05"), allStats.UserCount, allStats.TotalRPS, allStats.TotalFailRatio*100))
	println(fmt.Sprintf("Client Monitor: Cpu:%.1f%%, Memory:%.1f%%", computerMonitor.CPU, computerMonitor.Mem))
	table := tablewriter.NewWriter(os.Stdout)

	pTitle := "L90"
	if OutputOps.PercentTime > 0 {
		pTitle = fmt.Sprintf("L%d", OutputOps.PercentTime)
	}

	table.SetHeader([]string{"Type", "Name", "# requests", "# fails", "L50", pTitle, "Average", "Min", "Max", "Content Size", "# reqs/sec", "# fails/sec"})

	table.Append([]string{"Current Data:"})
	for _, stat := range output.Stats {
		row := make([]string, 12)
		row[0] = stat.Method
		row[1] = stat.Name
		row[2] = strconv.FormatInt(stat.NumRequests, 10)
		row[3] = strconv.FormatInt(stat.NumFailures, 10)
		row[4] = strconv.FormatInt(stat.medianResponseTime, 10)
		row[5] = strconv.FormatInt(stat.percentResponseTime, 10)
		row[6] = strconv.FormatFloat(stat.avgResponseTime, 'f', 2, 64)
		row[7] = strconv.FormatInt(stat.MinResponseTime, 10)
		row[8] = strconv.FormatInt(stat.MaxResponseTime, 10)
		row[9] = strconv.FormatInt(stat.avgContentLength, 10)
		row[10] = strconv.FormatInt(stat.currentRps, 10)
		row[11] = strconv.FormatInt(stat.currentFailPerSec, 10)
		table.Append(row)
	}

	table.Append([]string{"Summary Data:"})

	for _, stat := range allStats.Stats {
		row := make([]string, 12)
		row[0] = stat.Method
		row[1] = stat.Name
		row[2] = strconv.FormatInt(stat.NumRequests, 10)
		row[3] = strconv.FormatInt(stat.NumFailures, 10)
		row[4] = strconv.FormatInt(stat.medianResponseTime, 10)
		row[5] = strconv.FormatInt(stat.percentResponseTime, 10)
		row[6] = strconv.FormatFloat(stat.avgResponseTime, 'f', 2, 64)
		row[7] = strconv.FormatInt(stat.MinResponseTime, 10)
		row[8] = strconv.FormatInt(stat.MaxResponseTime, 10)
		row[9] = strconv.FormatInt(stat.avgContentLength, 10)
		row[10] = strconv.FormatInt(stat.currentRps, 10)
		row[11] = strconv.FormatInt(stat.currentFailPerSec, 10)
		table.Append(row)
	}
	table.Render()
	println()

}

var allStats *dataOutput

func buildAllStats(output *dataOutput) {
	if allStats == nil {
		allStats = new(dataOutput)
	}
	allStats.UserCount = output.UserCount

	for _, oItem := range output.Stats {
		hasItem := false
		for _, aItem := range allStats.Stats {
			if oItem.Name == aItem.Name {
				aItem.NumRequests = aItem.NumRequests + oItem.NumRequests
				aItem.NumFailures = aItem.NumFailures + oItem.NumFailures
				if oItem.MaxResponseTime > aItem.MaxResponseTime {
					aItem.MaxResponseTime = oItem.MaxResponseTime
				}
				if oItem.MinResponseTime < aItem.MinResponseTime {
					aItem.MinResponseTime = oItem.MinResponseTime
				}
				aItem.TotalResponseTime = aItem.TotalResponseTime + oItem.TotalResponseTime
				aItem.TotalContentLength = aItem.TotalContentLength + oItem.TotalContentLength
				aItem.avgContentLength = aItem.TotalContentLength / aItem.NumRequests
				for key, value := range oItem.ResponseTimes {
					if _, ok := aItem.ResponseTimes[key]; ok {
						aItem.ResponseTimes[key] = aItem.ResponseTimes[key] + value
					} else {
						aItem.ResponseTimes[key] = value
					}
				}
				for key, value := range oItem.NumReqsPerSec {
					if _, ok := aItem.NumReqsPerSec[key]; ok {
						aItem.NumReqsPerSec[key] = aItem.NumReqsPerSec[key] + value
					} else {
						aItem.NumReqsPerSec[key] = value
					}
				}
				for key, value := range oItem.NumFailPerSec {
					aItem.NumFailPerSec[key] = value
				}
				aItem.medianResponseTime = getMedianResponseTime(aItem.NumRequests, aItem.ResponseTimes)
				aItem.percentResponseTime = getPercentResponseTime(aItem.NumRequests, aItem.ResponseTimes)
				aItem.avgResponseTime = getAvgResponseTime(aItem.NumRequests, aItem.TotalResponseTime)
				aItem.currentRps = getCurrentRps(aItem.NumRequests, aItem.NumReqsPerSec)
				aItem.currentFailPerSec = getCurrentFailPerSec(aItem.NumFailures, aItem.NumFailPerSec)

				hasItem = true
				break
			}
		}

		if !hasItem {
			allStats.Stats = append(allStats.Stats, oItem)
		}
	}

	if allStats.NumReqsPerSec == nil {
		allStats.NumReqsPerSec = map[int64]int64{}
	}

	allStats.TotalRPS = 0
	allStats.TotalRequestCount = 0
	allStats.TotalFailedCount = 0
	for _, aItem := range allStats.Stats {
		allStats.TotalRPS = allStats.TotalRPS + aItem.currentRps
		allStats.TotalRequestCount = allStats.TotalRequestCount + aItem.NumRequests
		allStats.TotalFailedCount = allStats.TotalFailedCount + aItem.NumFailures
	}

	if allStats.TotalRequestCount != 0 {
		allStats.TotalFailRatio = float64(allStats.TotalFailedCount) / float64(allStats.TotalRequestCount)
	}
}

func sortOutput(stats []*statsEntryOutput) []*statsEntryOutput {
	if stats == nil {
		return nil
	}
	for i := 0; i < len(stats); i++ {
		for j := 1; j < len(stats)-i; j++ {
			if stats[j].Name < stats[j-1].Name {
				tmp := stats[j-1]
				stats[j-1] = stats[j]
				stats[j] = tmp
			}
		}
	}
	return stats
}

type statsEntryOutput struct {
	statsEntry

	medianResponseTime  int64   // median response time
	percentResponseTime int64   // custom a percent response time
	avgResponseTime     float64 // average response time, round float to 2 decimal places
	avgContentLength    int64   // average content size
	currentRps          int64   // # reqs/sec
	currentFailPerSec   int64   // # fails/sec
}

type dataOutput struct {
	UserCount      int32               `json:"user_count"`
	TotalStats     *statsEntryOutput   `json:"stats_total"`
	TotalRPS       int64               `json:"total_rps"`
	TotalFailRatio float64             `json:"total_fail_ratio"`
	Stats          []*statsEntryOutput `json:"stats"`
	//AllTotalStats  []*statsEntryOutput               `json:"all_total_stats"`
	Errors map[string]map[string]interface{} `json:"errors"`

	TotalRequestCount int64
	TotalFailedCount  int64
	NumReqsPerSec     map[int64]int64
}

func convertData(data map[string]interface{}) (output *dataOutput, err error) {
	userCount, ok := data["user_count"].(int32)
	if !ok {
		return nil, fmt.Errorf("user_count is not int32")
	}
	stats, ok := data["stats"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("stats is not []interface{}")
	}

	// convert stats in total
	statsTotal, ok := data["stats_total"].(interface{})
	if !ok {
		return nil, fmt.Errorf("stats_total is not interface{}")
	}
	entryTotalOutput, err := deserializeStatsEntry(statsTotal)
	if err != nil {
		return nil, err
	}

	output = &dataOutput{
		UserCount:      userCount,
		TotalStats:     entryTotalOutput,
		TotalRPS:       getCurrentRps(entryTotalOutput.NumRequests, entryTotalOutput.NumReqsPerSec),
		TotalFailRatio: getTotalFailRatio(entryTotalOutput.NumRequests, entryTotalOutput.NumFailures),
		Stats:          make([]*statsEntryOutput, 0, len(stats)),
		NumReqsPerSec:  entryTotalOutput.NumReqsPerSec,
	}

	// convert stats
	for _, stat := range stats {
		entryOutput, err := deserializeStatsEntry(stat)
		if err != nil {
			return nil, err
		}
		output.Stats = append(output.Stats, entryOutput)
	}
	return
}

func deserializeStatsEntry(stat interface{}) (entryOutput *statsEntryOutput, err error) {
	statBytes, err := json.Marshal(stat)
	if err != nil {
		return nil, err
	}
	entry := statsEntry{}
	if err = json.Unmarshal(statBytes, &entry); err != nil {
		return nil, err
	}

	numRequests := entry.NumRequests
	entryOutput = &statsEntryOutput{
		statsEntry:          entry,
		medianResponseTime:  getMedianResponseTime(numRequests, entry.ResponseTimes),
		percentResponseTime: getPercentResponseTime(numRequests, entry.ResponseTimes),
		avgResponseTime:     getAvgResponseTime(numRequests, entry.TotalResponseTime),
		avgContentLength:    getAvgContentLength(numRequests, entry.TotalContentLength),
		currentRps:          getCurrentRps(numRequests, entry.NumReqsPerSec),
		currentFailPerSec:   getCurrentFailPerSec(entry.NumFailures, entry.NumFailPerSec),
	}
	return
}

const (
	namespace = "boomer"
)

// gauge vectors for requests
var (
	gaugeNumRequests = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "num_requests",
			Help:      "The number of requests",
		},
		[]string{"method", "name"},
	)
	gaugeNumFailures = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "num_failures",
			Help:      "The number of failures",
		},
		[]string{"method", "name"},
	)
	gaugeMedianResponseTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "median_response_time",
			Help:      "The median response time",
		},
		[]string{"method", "name"},
	)
	gaugeAverageResponseTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "average_response_time",
			Help:      "The average response time",
		},
		[]string{"method", "name"},
	)
	gaugeMinResponseTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "min_response_time",
			Help:      "The min response time",
		},
		[]string{"method", "name"},
	)
	gaugeMaxResponseTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "max_response_time",
			Help:      "The max response time",
		},
		[]string{"method", "name"},
	)
	gaugeAverageContentLength = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "average_content_length",
			Help:      "The average content length",
		},
		[]string{"method", "name"},
	)
	gaugeCurrentRPS = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "current_rps",
			Help:      "The current requests per second",
		},
		[]string{"method", "name"},
	)
	gaugeCurrentFailPerSec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "current_fail_per_sec",
			Help:      "The current failure number per second",
		},
		[]string{"method", "name"},
	)
)

// gauges for total
var (
	gaugeUsers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "users",
			Help:      "The current number of users",
		},
	)
	gaugeTotalRPS = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "total_rps",
			Help:      "The requests per second in total",
		},
	)
	gaugeTotalFailRatio = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "fail_ratio",
			Help:      "The ratio of request failures in total",
		},
	)
)

// NewPrometheusPusherOutput returns a PrometheusPusherOutput.
func NewPrometheusPusherOutput(gatewayURL, jobName string) *PrometheusPusherOutput {
	return &PrometheusPusherOutput{
		pusher: push.New(gatewayURL, jobName),
	}
}

// PrometheusPusherOutput pushes boomer stats to Prometheus Pushgateway.
type PrometheusPusherOutput struct {
	pusher *push.Pusher // Prometheus Pushgateway Pusher
}

// OnStart will register all prometheus metric collectors
func (o *PrometheusPusherOutput) OnStart() {
	log.Println("register prometheus metric collectors")
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		// gauge vectors for requests
		gaugeNumRequests,
		gaugeNumFailures,
		gaugeMedianResponseTime,
		gaugeAverageResponseTime,
		gaugeMinResponseTime,
		gaugeMaxResponseTime,
		gaugeAverageContentLength,
		gaugeCurrentRPS,
		gaugeCurrentFailPerSec,
		// gauges for total
		gaugeUsers,
		gaugeTotalRPS,
		gaugeTotalFailRatio,
	)
	o.pusher = o.pusher.Gatherer(registry)
}

// OnStop of PrometheusPusherOutput has nothing to do.
func (o *PrometheusPusherOutput) OnStop() {

}

// OnEvent will push metric to Prometheus Pushgataway
func (o *PrometheusPusherOutput) OnEvent(data map[string]interface{}) {
	output, err := convertData(data)
	if err != nil {
		log.Println(fmt.Sprintf("convert data error: %v", err))
		return
	}

	// user count
	gaugeUsers.Set(float64(output.UserCount))

	// rps in total
	gaugeTotalRPS.Set(float64(output.TotalRPS))

	// failure ratio in total
	gaugeTotalFailRatio.Set(output.TotalFailRatio)

	for _, stat := range output.Stats {
		method := stat.Method
		name := stat.Name
		gaugeNumRequests.WithLabelValues(method, name).Set(float64(stat.NumRequests))
		gaugeNumFailures.WithLabelValues(method, name).Set(float64(stat.NumFailures))
		gaugeMedianResponseTime.WithLabelValues(method, name).Set(float64(stat.medianResponseTime))
		gaugeAverageResponseTime.WithLabelValues(method, name).Set(float64(stat.avgResponseTime))
		gaugeMinResponseTime.WithLabelValues(method, name).Set(float64(stat.MinResponseTime))
		gaugeMaxResponseTime.WithLabelValues(method, name).Set(float64(stat.MaxResponseTime))
		gaugeAverageContentLength.WithLabelValues(method, name).Set(float64(stat.avgContentLength))
		gaugeCurrentRPS.WithLabelValues(method, name).Set(float64(stat.currentRps))
		gaugeCurrentFailPerSec.WithLabelValues(method, name).Set(float64(stat.currentFailPerSec))
	}

	if err := o.pusher.Push(); err != nil {
		log.Println(fmt.Sprintf("Could not push to Pushgateway: error: %v", err))
	}
}
