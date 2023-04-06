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
	PercentTime        int
	RealTimeResultPath string
	TotalResultPath    string
}

func NewConsoleOutputWithOptions(outputOps *OutputOptions) *ConsoleOutput {
	OutputOps = outputOps
	if OutputOps.PercentTime <= 0 {
		OutputOps.PercentTime = 90
	}
	return &ConsoleOutput{}
}

func NewJsonFileOutputWithOptions(outputOps *OutputOptions) *JsonFileOutput {
	OutputOps = outputOps
	if OutputOps.PercentTime <= 0 {
		OutputOps.PercentTime = 90
	}
	return &JsonFileOutput{}
}

var OutputOps = new(OutputOptions)

// NewConsoleOutput returns a ConsoleOutput.
func NewConsoleOutput() *ConsoleOutput {
	return &ConsoleOutput{}
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
		row[4] = strconv.FormatInt(stat.MedianResponseTime, 10)
		row[5] = strconv.FormatInt(stat.PercentResponseTime, 10)
		row[6] = strconv.FormatFloat(stat.AvgResponseTime, 'f', 2, 64)
		row[7] = strconv.FormatInt(stat.MinResponseTime, 10)
		row[8] = strconv.FormatInt(stat.MaxResponseTime, 10)
		row[9] = strconv.FormatInt(stat.AvgContentLength, 10)
		row[10] = strconv.FormatInt(stat.CurrentRps, 10)
		row[11] = strconv.FormatInt(stat.CurrentFailPerSec, 10)
		table.Append(row)
	}

	table.Append([]string{"Summary Data:"})

	for _, stat := range allStats.Stats {
		row := make([]string, 12)
		row[0] = stat.Method
		row[1] = stat.Name
		row[2] = strconv.FormatInt(stat.NumRequests, 10)
		row[3] = strconv.FormatInt(stat.NumFailures, 10)
		row[4] = strconv.FormatInt(stat.MedianResponseTime, 10)
		row[5] = strconv.FormatInt(stat.PercentResponseTime, 10)
		row[6] = strconv.FormatFloat(stat.AvgResponseTime, 'f', 2, 64)
		row[7] = strconv.FormatInt(stat.MinResponseTime, 10)
		row[8] = strconv.FormatInt(stat.MaxResponseTime, 10)
		row[9] = strconv.FormatInt(stat.AvgContentLength, 10)
		row[10] = strconv.FormatInt(stat.CurrentRps, 10)
		row[11] = strconv.FormatInt(stat.CurrentFailPerSec, 10)
		table.Append(row)
	}
	table.Render()
	println()
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
	var count int64
	if len(numReqsPerSecond) > 2 {
		count = int64(len(numReqsPerSecond) - 1)
	} else {
		count = int64(len(numReqsPerSecond))
	}
	//numReqsPerSecondLength := int64(len(numReqsPerSecond))
	//if numReqsPerSecondLength < 15 {
	//	for key, value := range numReqsPerSecond {
	//		fmt.Printf("[%d:%d]", key, value)
	//	}
	//}
	if count != 0 {
		currentRps = numRequests / count
	}
	//CurrentRps = numRequests / int64(len(numReqsPerSecond))
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

type JsonFileOutput struct {
}

func NewJsonFileOutput() *JsonFileOutput {
	return &JsonFileOutput{}
}

func (o *JsonFileOutput) OnStart() {

}

func (o *JsonFileOutput) OnStop() {

}

func (o *JsonFileOutput) OnEvent(data map[string]interface{}) {
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

	// 当前统计数据
	if OutputOps.RealTimeResultPath != "" {
		realTimeResult := ResultData{
			UserCount:      output.UserCount,
			TotalRPS:       output.TotalRPS,
			TotalFailRatio: output.TotalFailRatio,
			//Errors:         output.Errors,
		}
		if output.TotalStats != nil {
			realTimeResult.TotalStats = *output.TotalStats
			realTimeResult.TotalStats.NumReqsPerSec = nil
			realTimeResult.TotalStats.NumFailPerSec = nil
			realTimeResult.TotalStats.ResponseTimes = nil
		}
		if output.Stats != nil {
			realTimeResult.Stats = []statsEntryOutput{}
			for _, stat := range output.Stats {
				temStat := *stat
				temStat.NumFailPerSec = nil
				temStat.NumReqsPerSec = nil
				temStat.ResponseTimes = nil
				realTimeResult.Stats = append(realTimeResult.Stats, temStat)
			}
		}
		jsonOutPut := RealTimeJsonOutput{
			CurrTime:   currentTime,
			CurrResult: realTimeResult,
			Monitor:    computerMonitor,
		}
		jsonStr, _ := json.Marshal(jsonOutPut)
		AppendLineToFile(OutputOps.RealTimeResultPath, string(jsonStr))
	}

	// 累计统计数据
	if OutputOps.TotalResultPath != "" && allStats != nil {
		totalResult := ResultData{
			UserCount:      allStats.UserCount,
			TotalRPS:       allStats.TotalRPS,
			TotalFailRatio: allStats.TotalFailRatio,
			Errors:         allStats.Errors,
		}
		if allStats.TotalStats != nil {
			totalResult.TotalStats = *allStats.TotalStats
			totalResult.TotalStats.NumFailPerSec = nil
			totalResult.TotalStats.NumReqsPerSec = nil
			totalResult.TotalStats.ResponseTimes = nil
		}
		if output.Stats != nil {
			totalResult.Stats = []statsEntryOutput{}
			for _, stat := range output.Stats {
				temStat := *stat
				temStat.NumFailPerSec = nil
				temStat.NumReqsPerSec = nil
				temStat.ResponseTimes = nil
				totalResult.Stats = append(totalResult.Stats, temStat)
			}
		}
		totalOutput := TotalJsonOutput{
			TotalData: totalResult,
			CurrTime:  currentTime,
		}
		jsonStr, _ := json.Marshal(totalOutput)
		WriteTextToFile(OutputOps.TotalResultPath, string(jsonStr))
	}
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
				if aItem.NumRequests > 0 {
					aItem.AvgContentLength = aItem.TotalContentLength / aItem.NumRequests
				}
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
				aItem.MedianResponseTime = getMedianResponseTime(aItem.NumRequests, aItem.ResponseTimes)
				aItem.PercentResponseTime = getPercentResponseTime(aItem.NumRequests, aItem.ResponseTimes)
				aItem.AvgResponseTime = getAvgResponseTime(aItem.NumRequests, aItem.TotalResponseTime)
				aItem.CurrentRps = getCurrentRps(aItem.NumRequests, aItem.NumReqsPerSec)
				aItem.CurrentFailPerSec = getCurrentFailPerSec(aItem.NumFailures, aItem.NumFailPerSec)
				hasItem = true
				break
			}
		}

		if !hasItem {
			allStats.Stats = append(allStats.Stats, oItem)
		}
	}

	if allStats.TotalStats == nil {
		allStats.TotalStats = &statsEntryOutput{
			statsEntry: statsEntry{
				Name: "Total",
			},
		}
	} else {
		allStats.TotalStats.NumRequests += output.TotalStats.NumRequests
		allStats.TotalStats.NumFailures += output.TotalStats.NumFailures
	}

	if allStats.Errors == nil {
		allStats.Errors = map[string]map[string]interface{}{}
	}
	for key, value := range output.Errors {
		hasAdd := false
		for key1, value1 := range allStats.Errors {
			if key == key1 {
				value1["occurrences"] = value1["occurrences"].(int64) + value["occurrences"].(int64)
				hasAdd = true
				break
			}
		}
		if !hasAdd {
			allStats.Errors[key] = value
		}
	}

	if allStats.NumReqsPerSec == nil {
		allStats.NumReqsPerSec = map[int64]int64{}
	}

	allStats.TotalRPS = 0
	//allStats.TotalRequestCount = 0
	//allStats.TotalFailedCount = 0
	for _, aItem := range allStats.Stats {
		allStats.TotalRPS = allStats.TotalRPS + aItem.CurrentRps
		allStats.TotalRequestCount = allStats.TotalRequestCount + aItem.NumRequests
		allStats.TotalFailedCount = allStats.TotalFailedCount + aItem.NumFailures
	}

	//allStats.TotalStats
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

	MedianResponseTime  int64   `json:"median_response_time"`  // median response time
	PercentResponseTime int64   `json:"percent_response_time"` // custom a percent response time
	AvgResponseTime     float64 `json:"avg_response_time"`     // average response time, round float to 2 decimal places
	AvgContentLength    int64   `json:"avg_content_length"`    // average content size
	CurrentRps          int64   `json:"current_rps"`           // # reqs/sec
	CurrentFailPerSec   int64   `json:"current_fail_per_sec"`  // # fails/sec
}

type dataOutput struct {
	UserCount      int32                             `json:"user_count"`
	TotalStats     *statsEntryOutput                 `json:"stats_total"`
	TotalRPS       int64                             `json:"total_rps"`
	TotalFailRatio float64                           `json:"total_fail_ratio"`
	Stats          []*statsEntryOutput               `json:"stats"`
	Errors         map[string]map[string]interface{} `json:"errors"`

	TotalRequestCount int64           `json:"-"`
	TotalFailedCount  int64           `json:"-"`
	NumReqsPerSec     map[int64]int64 `json:"-"`
}
type DataOutputJson struct {
	UserCount int32
}
type RealTimeJsonOutput struct {
	Monitor    ComputerMonitor `json:"monitor"`
	CurrResult ResultData      `json:"curr_result"`
	CurrTime   time.Time       `json:"curr_time"`
}
type TotalJsonOutput struct {
	TotalData ResultData `json:"total_result"`
	CurrTime  time.Time  `json:"curr_time""`
}

type ResultData struct {
	UserCount      int32                             `json:"user_count"`
	TotalRPS       int64                             `json:"total_rps"`
	TotalFailRatio float64                           `json:"total_fail_ratio"`
	TotalStats     statsEntryOutput                  `json:"stats_total"`
	Stats          []statsEntryOutput                `json:"stats"`
	Errors         map[string]map[string]interface{} `json:"errors"`
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
	errors := data["errors"].(map[string]map[string]interface{})

	output = &dataOutput{
		UserCount:      userCount,
		TotalStats:     entryTotalOutput,
		TotalRPS:       getCurrentRps(entryTotalOutput.NumRequests, entryTotalOutput.NumReqsPerSec),
		TotalFailRatio: getTotalFailRatio(entryTotalOutput.NumRequests, entryTotalOutput.NumFailures),
		Stats:          make([]*statsEntryOutput, 0, len(stats)),
		NumReqsPerSec:  entryTotalOutput.NumReqsPerSec,
		Errors:         errors,
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
		MedianResponseTime:  getMedianResponseTime(numRequests, entry.ResponseTimes),
		PercentResponseTime: getPercentResponseTime(numRequests, entry.ResponseTimes),
		AvgResponseTime:     getAvgResponseTime(numRequests, entry.TotalResponseTime),
		AvgContentLength:    getAvgContentLength(numRequests, entry.TotalContentLength),
		CurrentRps:          getCurrentRps(numRequests, entry.NumReqsPerSec),
		CurrentFailPerSec:   getCurrentFailPerSec(entry.NumFailures, entry.NumFailPerSec),
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
		gaugeMedianResponseTime.WithLabelValues(method, name).Set(float64(stat.MedianResponseTime))
		gaugeAverageResponseTime.WithLabelValues(method, name).Set(float64(stat.AvgResponseTime))
		gaugeMinResponseTime.WithLabelValues(method, name).Set(float64(stat.MinResponseTime))
		gaugeMaxResponseTime.WithLabelValues(method, name).Set(float64(stat.MaxResponseTime))
		gaugeAverageContentLength.WithLabelValues(method, name).Set(float64(stat.AvgContentLength))
		gaugeCurrentRPS.WithLabelValues(method, name).Set(float64(stat.CurrentRps))
		gaugeCurrentFailPerSec.WithLabelValues(method, name).Set(float64(stat.CurrentFailPerSec))
	}

	if err := o.pusher.Push(); err != nil {
		log.Println(fmt.Sprintf("Could not push to Pushgateway: error: %v", err))
	}
}
