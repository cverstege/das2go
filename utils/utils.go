package utils

// DAS utils module
//
// Copyright (c) 2015-2016 - Valentin Kuznetsov <vkuznet AT gmail dot com>
//

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// global variable for this module which we're going to use across many modules

// VERSION provides information about das2go build
var VERSION string

// VERBOSE holds level of verbosity, it is set in main
var VERBOSE int

// WEBSERVER declares that web server will run, it is set in main
var WEBSERVER int

// DASMAPS specifies location of dasmaps
var DASMAPS string

// Stack helper function to return Stack
func Stack() string {
	trace := make([]byte, 2048)
	count := runtime.Stack(trace, false)
	return fmt.Sprintf("\nStack of %d bytes: %s\n", count, trace)
}

// ErrPropagate error helper function which can be used in defer ErrPropagate()
func ErrPropagate(api string) {
	if err := recover(); err != nil {
		log.Fatalf("ERROR: api %v, error %v, Stack: %v\n", api, err, Stack())
	}
}

// ErrPropagate2Channel error helper function which can be used in goroutines as
// ch := make(chan interface{})
// go func() {
//    defer ErrPropagate2Channel(api, ch)
//    someFunction()
// }()
func ErrPropagate2Channel(api string, ch chan interface{}) {
	if err := recover(); err != nil {
		log.Printf("ERROR: api %v, error %v\n", api, Stack())
		ch <- fmt.Sprintf("%s:%s", api, err)
	}
}

// GoDeferFunc helper function to run any given function in defered go routine
func GoDeferFunc(api string, f func()) {
	ch := make(chan interface{})
	go func() {
		defer ErrPropagate2Channel(api, ch)
		f()
		ch <- "ok" // send to channel that we can read it later in case of success of f()
	}()
	err := <-ch
	if err != nil && err != "ok" {
		msg := fmt.Sprintf("unable to run api='%s', function='%#v', error='%v'", api, f, err)
		log.Fatal("ERROR ", msg)
	}
}

// FindInList helper function to find item in a list
func FindInList(a string, arr []string) bool {
	for _, e := range arr {
		if e == a {
			return true
		}
	}
	return false
}

// InList helper function to check item in a list
func InList(a string, list []string) bool {
	check := 0
	for _, b := range list {
		if b == a {
			check += 1
		}
	}
	if check != 0 {
		return true
	}
	return false
}

// MapKeys helper function to return keys from a map
func MapKeys(rec map[string]interface{}) []string {
	keys := make([]string, 0, len(rec))
	for k := range rec {
		keys = append(keys, k)
	}
	return keys
}

// EqualLists helper function to compare list of strings
func EqualLists(list1, list2 []string) bool {
	count := 0
	for _, k := range list1 {
		if InList(k, list2) {
			count += 1
		} else {
			return false
		}
	}
	if len(list2) == count {
		return true
	}
	return false
}

// CheckEntries helper function to check that entries from list1 are all appear in list2
func CheckEntries(list1, list2 []string) bool {
	var out []string
	for _, k := range list1 {
		if InList(k, list2) {
			//             count += 1
			out = append(out, k)
		}
	}
	if len(out) == len(list1) {
		return true
	}
	return false
}

// Expire helper function to convert expire timestamp (int) into seconds since epoch
func Expire(expire int) int64 {
	tstamp := strconv.Itoa(expire)
	if len(tstamp) == 10 {
		return int64(expire)
	}
	return int64(time.Now().Unix() + int64(expire))
}

// UnixTime helper function to convert given time into Unix timestamp
func UnixTime(ts string) int64 {
	// time is unix since epoch
	if len(ts) == 10 { // unix time
		tstamp, _ := strconv.ParseInt(ts, 10, 64)
		return tstamp
	}
	// YYYYMMDD, always use 2006 as year 01 for month and 02 for date since it is predefined int Go parser
	const layout = "20060102"
	t, err := time.Parse(layout, ts)
	if err != nil {
		log.Printf("ERROR: unable to parse ts %v error %v\n", ts, err)
		return 0
	}
	return int64(t.Unix())
}

// Unix2DASTime helper function to convert given time into Unix timestamp
func Unix2DASTime(ts int64) string {
	// YYYYMMDD, always use 2006 as year 01 for month and 02 for date since it is predefined int Go parser
	const layout = "20060102"
	t := time.Unix(ts, 0)
	return t.In(time.UTC).Format(layout)
}

// RunRegistryTime helper function to convert given time into RunRegistry timestamp
func RunRegistryTime(ts string) string {
	const runRegistryTime = "2006-01-02"
	// time is unix since epoch
	if len(ts) == 10 { // unix time
		tstamp, _ := strconv.ParseInt(ts, 10, 64)
		t := time.Unix(tstamp, 0)
		return t.Format(runRegistryTime)
	}
	// YYYYMMDD, always use 2006 as year 01 for month and 02 for date since it is predefined int Go parser
	const layout = "20060102"
	t, err := time.Parse(layout, ts)
	if err != nil {
		log.Printf("ERROR: unable to parse ts %v error %v\n", ts, err)
		return "N/A"
	}
	return t.Format(runRegistryTime)
}

// DashboardTime helper function to convert given time into Dashboard timestamp
func DashboardTime(ts string) string {
	const dashboardTime = "2006-01-02 15:04:05"
	// time is unix since epoch
	if len(ts) == 10 { // unix time
		tstamp, _ := strconv.ParseInt(ts, 10, 64)
		t := time.Unix(tstamp, 0)
		return t.Format(dashboardTime)
	}
	// YYYYMMDD, always use 2006 as year 01 for month and 02 for date since it is predefined int Go parser
	const layout = "20060102"
	t, err := time.Parse(layout, ts)
	if err != nil {
		log.Printf("ERROR: unable to parse ts %v error %v\n", ts, err)
		return "N/A"
	}
	return t.Format(dashboardTime)
}

// ConddbTime helper function to convert given time into Conddb timestamp
func ConddbTime(ts string) string {
	const conddbTime = "02-Jan-06-15:04"
	// time is unix since epoch
	if len(ts) == 10 { // unix time
		tstamp, _ := strconv.ParseInt(ts, 10, 64)
		t := time.Unix(tstamp, 0)
		return t.Format(conddbTime)
	}
	// YYYYMMDD, always use 2006 as year 01 for month and 02 for date since it is predefined int Go parser
	const layout = "20060102"
	t, err := time.Parse(layout, ts)
	if err != nil {
		log.Printf("ERROR: unable to parse ts %v error %v\n", ts, err)
		return "N/A"
	}
	return t.Format(conddbTime)
}

// List2Set helper function to convert input list into set
func List2Set(arr []string) []string {
	var out []string
	for _, key := range arr {
		if !InList(key, out) {
			out = append(out, key)
		}
	}
	return out
}

// TimeFormat helper function to convert Unix time into human readable form
func TimeFormat(ts interface{}) string {
	var err error
	var t int64
	switch v := ts.(type) {
	case int:
		t = int64(v)
	case int32:
		t = int64(v)
	case int64:
		t = v
	case float64:
		t = int64(v)
	case string:
		t, err = strconv.ParseInt(v, 0, 64)
		if err != nil {
			return fmt.Sprintf("%v", ts)
		}
	default:
		return fmt.Sprintf("%v", ts)
	}
	layout := "2006-01-02 15:04:05"
	return time.Unix(t, 0).UTC().Format(layout)
}

// SizeFormat helper function to convert size into human readable form
func SizeFormat(val interface{}) string {
	var size float64
	var err error
	switch v := val.(type) {
	case int:
		size = float64(v)
	case int32:
		size = float64(v)
	case int64:
		size = float64(v)
	case float64:
		size = v
	case string:
		size, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
	default:
		return fmt.Sprintf("%v", val)
	}
	base := 1000. // CMS convert is to use power of 10
	xlist := []string{"", "KB", "MB", "GB", "TB", "PB"}
	for _, vvv := range xlist {
		if size < base {
			return fmt.Sprintf("%v (%3.1f%s)", val, size, vvv)
		}
		size = size / base
	}
	return fmt.Sprintf("%v (%3.1f%s)", val, size, xlist[len(xlist)])
}

// IsInt helper function to test if given value is integer
func IsInt(val string) bool {
	return PatternInt.MatchString(val)
}

// Sum helper function to perform sum operation over provided array of values
func Sum(data []interface{}) float64 {
	out := 0.0
	for _, val := range data {
		if val != nil {
			//             out += val.(float64)
			switch v := val.(type) {
			case float64:
				out += v
			case json.Number:
				vv, e := v.Float64()
				if e == nil {
					out += vv
				}
			case int64:
				out += float64(v)
			}
		}
	}
	return out
}

// Max helper function to perform Max operation over provided array of values
func Max(data []interface{}) float64 {
	out := 0.0
	for _, val := range data {
		if val != nil {
			switch v := val.(type) {
			case float64:
				if v > out {
					out = v
				}
			case json.Number:
				vv, e := v.Float64()
				if e == nil && vv > out {
					out = vv
				}
			case int64:
				if float64(v) > out {
					out = float64(v)
				}
			}
		}
	}
	return out
}

// Min helper function to perform Min operation over provided array of values
func Min(data []interface{}) float64 {
	out := float64(^uint(0) >> 1) // largest int
	for _, val := range data {
		if val == nil {
			continue
		}
		switch v := val.(type) {
		case float64:
			if v < out {
				out = v
			}
		case json.Number:
			vv, e := v.Float64()
			if e == nil && vv < out {
				out = vv
			}
		case int64:
			if float64(v) < out {
				out = float64(v)
			}
		}
	}
	return out
}

// Mean helper function to perform Mean operation over provided array of values
func Mean(data []interface{}) float64 {
	return Sum(data) / float64(len(data))
}

// Avg helper function to perform Avg operation over provided array of values
func Avg(data []interface{}) float64 {
	return Mean(data)
}

// Median helper function to perform Median operation over provided array of values
func Median(data []interface{}) float64 {
	var input sort.Float64Slice
	var median float64
	for _, v := range data {
		switch val := v.(type) {
		case float64:
			input = append(input, val)
		case json.Number:
			vv, e := val.Float64()
			if e == nil {
				input = append(input, vv)
			}
		case int64:
			input = append(input, float64(val))
		}
	}
	input.Sort()
	l := len(input)
	if l == 0 {
		return 0
	} else if l%2 == 0 {
		median = (input[l/2-1] + input[l/2+1]) / 2.
	} else {
		median = float64(input[l/2])
	}
	return median
}

// IntList implement sort for []int type
type IntList []int

// Len provides length of the []int type
func (s IntList) Len() int { return len(s) }

// Swap implements swap function for []int type
func (s IntList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// Less implements less function for []int type
func (s IntList) Less(i, j int) bool { return s[i] < s[j] }

// Int64List implement sort for []int type
type Int64List []int64

// Len provides length of the []int64 type
func (s Int64List) Len() int { return len(s) }

// Swap implements swap function for []int64 type
func (s Int64List) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// Less implements less function for []int64 type
func (s Int64List) Less(i, j int) bool { return s[i] < s[j] }

// StringList implement sort for []string type
type StringList []string

// Len provides length of the []int type
func (s StringList) Len() int { return len(s) }

// Swap implements swap function for []int type
func (s StringList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// Less implements less function for []int type
func (s StringList) Less(i, j int) bool { return s[i] < s[j] }

// GetBytes converts interface to bytes
func GetBytes(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GetEnv fetches value from user environement
func GetEnv(key string) string {
	for _, item := range os.Environ() {
		value := strings.Split(item, "=")
		if value[0] == key {
			return value[1]
		}
	}
	return ""
}

// LoadExamples loads DAS examples from github or local file
func LoadExamples(ename, home string) string {
	githubUrl := fmt.Sprintf("https://raw.githubusercontent.com/dmwm/das2go/master/examples/%s", ename)
	if home == "" {
		for _, item := range os.Environ() {
			value := strings.Split(item, "=")
			if value[0] == "HOME" {
				home = value[1]
				break
			}
		}
	}
	dname := fmt.Sprintf("%s/.dasexamples", home)
	if _, err := os.Stat(dname); err != nil {
		os.Mkdir(dname, 0777)
	}
	fname := fmt.Sprintf("%s/.dasexamples/%s", home, ename)
	client := HttpClient()
	if _, err := os.Stat(fname); err != nil {
		// download maps from github
		resp := FetchResponse(client, githubUrl, "")
		if resp.Error == nil {
			// write data to local area
			err := os.WriteFile(fname, []byte(resp.Data), 0777)
			if err != nil {
				log.Println("ERROR: unable to write DAS example file", err)
				return ""
			}
		} else {
			log.Println("ERROR: unable to get DAS example file from github", resp.Error)
			return ""
		}
	}
	data, err := os.ReadFile(fname)
	if err != nil {
		log.Printf("ERROR: unable to read DAS example file %v, error %v\n", fname, err)
		return ""
	}
	return string(data)
}

// GetHostUrl return URL of given URL path
func GetHostUrl(s string) string {
	arr := strings.Split(s, "://")
	if len(arr) > 1 {
		hurl := strings.Split(arr[1], "/")
		return fmt.Sprintf("%s://%s", arr[0], hurl[0])
	}
	return ""
}

// Color prints given string in color based on ANSI escape codes, see
// http://www.wikiwand.com/en/ANSI_escape_code#/Colors
func Color(col, text string) string {
	return BOLD + "\x1b[" + col + text + PLAIN
}

// ColorUrl returns colored string of given url
func ColorUrl(rurl string) string {
	return Color(BLUE, rurl)
}

// DASError prints DAS error message with given arguments
func DASError(args ...interface{}) {
	fmt.Println(Color(RED, "DAS ERROR"), args)
}

// DASWarning prints DAS error message with given arguments
func DASWarning(args ...interface{}) {
	fmt.Println(Color(BROWN, "DAS WARNING"), args)
}

// BLACK color
const BLACK = "0;30m"

// RED color
const RED = "0;31m"

// GREEN color
const GREEN = "0;32m"

// BROWN color
const BROWN = "0;33m"

// BLUE color
const BLUE = "0;34m"

// PURPLE color
const PURPLE = "0;35m"

// CYAN color
const CYAN = "0;36m"

// LIGHT_PURPLE color
const LIGHT_PURPLE = "1;35m"

// LIGHT_CYAN color
const LIGHT_CYAN = "1;36m"

// BOLD type
const BOLD = "\x1b[1m"

// PLAIN type
const PLAIN = "\x1b[0m"
