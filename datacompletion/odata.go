package datacompletion

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pawelmarkowski/gocloak-retry-channels/keycloak"
)

// https://docs.microsoft.com/en-us/odata/client/query-options
type odata struct {
	source
	keycloak   *keycloak.TokenJWT
	pagination paginationMethod
	jobsChan   chan *url.URL
	returnChan chan []string
}

type paginationMethod interface {
	next(urlStr string, results int) (string, error)
}

type offsetPagination struct {
}

func (op *offsetPagination) next(urlStr string, results int) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	skip, err := strconv.Atoi(u.Query().Get("$skip"))

	return strings.Replace(urlStr,
		"skip="+strconv.Itoa(skip),
		"skip="+strconv.Itoa(skip+results), 1), nil
}

// https://mholt.github.io/json-to-go/
type odataResp struct {
	Value []struct {
		Name string `json:"name"`
	} `json:"value"`
}

type Retry struct {
	nums      int
	transport http.RoundTripper
}

func (r *Retry) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	for i := 0; i < r.nums; i++ {
		resp, err = r.transport.RoundTrip(req)
		if resp != nil && err == nil {
			return
		}
		log.Println("Retrying http request... Attempt:", i+1)
		time.Sleep(time.Duration(4+rand.Intn(5)) * time.Second)
	}
	return
}

var client = &http.Client{Timeout: 60 * time.Second, Transport: &Retry{
	nums:      5,
	transport: http.DefaultTransport,
}}

func (o *odata) loadData(ctx context.Context, ch chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("started worker")
	var err error
	for urlStr := range ch {
		retryCounter := 0
		for {
			fmt.Println(urlStr)
			req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
			if err != nil {
				break
			}
			req.Header.Add("Authorization", "Bearer "+o.keycloak.GetToken().AccessToken)
			resp, err := client.Do(req)
			if err != nil {
				retryCounter += 1
				fmt.Println("Request error", err, "(retry #", retryCounter, ")")
				if retryCounter < 3 {
					time.Sleep(time.Duration(4+rand.Intn(5)) * time.Second)
					continue
				} else {
					break
				}
			}
			parsedResp := &odataResp{}
			err = json.NewDecoder(resp.Body).Decode(parsedResp)
			resp.Body.Close()
			if err != nil {
				retryCounter += 1
				fmt.Println("Parsing resp error", err, "(retry #", retryCounter, ")")
				if retryCounter < 3 {
					time.Sleep(time.Duration(4+rand.Intn(5)) * time.Second)
					continue
				} else {
					break
				}
			}
			for item := range parsedResp.Value {
				o.source.items.Store(parsedResp.Value[item].Name, true)
			}
			urlStr, err = o.pagination.next(urlStr, len(parsedResp.Value))
			if len(parsedResp.Value) == 0 {
				break
			}
		}
		if errors.Is(err, context.Canceled) {
			break
		}
	}
	defer fmt.Println("closed worker")
}

func (o *odata) setFilter() {
	fmt.Println("set filter")
}

func (o *odata) setPagination(urlStr string) error {
	if strings.Contains(urlStr, "skiptoken") {
		return fmt.Errorf("skip toen not supported")
	} else if strings.Contains(urlStr, "$skip") {
		o.pagination = &offsetPagination{}
		return nil
	} else {
		return fmt.Errorf("Can not match pagination method")
	}
}

// TODO add ctx for gracefull shutdown
func newOdata(ctx context.Context, kc *keycloak.TokenJWT, urlStr string, wg *sync.WaitGroup, ch chan [][]string) {
	defer wg.Done()
	compRegex := regexp.MustCompile(`[\.,\ \/\:]`)
	outputCsv := compRegex.ReplaceAllString(urlStr, "_") + ".csv"
	urlStr = strings.ReplaceAll(urlStr, " ", "%20")
	odata := &odata{
		source:   source{},
		keycloak: kc,
	}
	odata.setPagination(urlStr)

	loadWg := &sync.WaitGroup{}
	jobsChan := make(chan string)
	workers, _ := strconv.Atoi(os.Getenv("SOURCE_WORKERS"))
	loadWg.Add(workers)
	for i := 0; i < workers; i++ {
		go odata.loadData(ctx, jobsChan, loadWg)
	}
	if strings.Contains(urlStr, "date%20ge") {
		splitToMultipleJobsByDate(jobsChan, urlStr)
	} else {
		jobsChan <- urlStr
	}
	close(jobsChan)
	loadWg.Wait()
	keys := make([][]string, len(odata.source.GetItems()))
	i := 0
	for k := range odata.source.GetItems() {
		keys[i] = []string{k}
		i++
	}
	SaveResults(outputCsv, keys)
	ch <- keys
}

func SaveResults(filename string, records [][]string) {
	f, err := os.Create(filename)
	defer f.Close()

	if err != nil {
		log.Fatalln("failed to open file", err)
	}

	w := csv.NewWriter(f)
	err = w.WriteAll(records)
	if err != nil {
		log.Fatal(err)
	}
}

func splitToMultipleJobsByDate(jobsChan chan string, urlStr string) {
	regExpr := regexp.MustCompile(`date%20ge%20(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}[^Z]*Z)`)
	dateFromStr := regExpr.FindStringSubmatch(urlStr)[1]
	dateFrom, err := time.Parse(time.RFC3339, dateFromStr)
	if err != nil {
		return
	}
	regExpr = regexp.MustCompile(`date%20lt%20(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}[^Z]*Z)`)
	dateToStr := regExpr.FindStringSubmatch(urlStr)[1]
	dateTo, err := time.Parse(time.RFC3339, dateToStr)
	if err != nil {
		dateTo = time.Now()
	}
	fmt.Println("Spliting tasks for daterange", dateFrom, "/", dateTo)
	for dateFrom.Before(dateTo.Add(-1 * time.Minute)) {
		jobsChan <- strings.Replace(
			strings.Replace(
				urlStr,
				dateToStr,
				dateFrom.Add(1*time.Minute).Format("2006-01-02T15:04:05.000Z"),
				1),
			dateFromStr,
			dateFrom.Format("2006-01-02T15:04:05.000Z"),
			1)
		dateFrom = dateFrom.Add(1 * time.Minute)
	}
	jobsChan <- strings.Replace(urlStr, dateFromStr, dateFrom.Format("2006-01-02T15:04:05.000Z"), 1)
}
