// Copyright 2017 Kumina, https://kumina.nl/
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	deletedMappingsUpDesc = prometheus.NewDesc(
		prometheus.BuildFQName("old_lib_exporter", "", "up"),
		"Whether scraping old lib's metrics was successful.",
		nil,
		nil)
)

func searchMaps(reader io.Reader, collection *map[string]int) error {
	scanner := bufio.NewScanner(reader)
	re := regexp.MustCompile(" [-r][-w]xp .* (/.*) \\(deleted\\)$")
	for scanner.Scan() {
		fields := re.FindStringSubmatch(scanner.Text())
		if fields != nil {
			_, ok := (*collection)[fields[1]]
			if ok {
				(*collection)[fields[1]] += 1
			} else {
				(*collection)[fields[1]] = 1
			}
		}
	}
	return scanner.Err()
}

func CollectDeletedMappings(procPath string, ch chan<- prometheus.Metric) error {
	collection := map[string]int{}
	files, err := ioutil.ReadDir(procPath)
	if err != nil {
		panic(err)
	}

	var scrapeError error = nil
	for _, file := range files {
		_, err := strconv.Atoi(file.Name())
		if err == nil {
			read, err := os.Open(path.Join("/proc", file.Name(), "maps"))
			if err != nil {
				scrapeError = err
			} else {
				err = searchMaps(read, &collection)
				if err != nil {
					scrapeError = err
				}
			}
		}
	}

	deletedMappings := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "old_lib_exporter",
			Name:      "oldlibs",
			Help:      "Number of processes using this old library.",
		},
		[]string{"libraryname"},
	)

	for key, value := range collection {
		fmt.Println("Key:", key, "Value:", value)
		deletedMappings.WithLabelValues(key).Set(float64(value))
	}
	deletedMappings.Collect(ch)
	return scrapeError
}

type DeletedMappingExporter struct {
	procPath string
}

func NewDeletedMappingExporter(procPath string) (*DeletedMappingExporter, error) {
	return &DeletedMappingExporter{
		procPath: procPath,
	}, nil
}

func (e *DeletedMappingExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- deletedMappingsUpDesc
}

func (e *DeletedMappingExporter) Collect(ch chan<- prometheus.Metric) {
	err := CollectDeletedMappings(e.procPath, ch)
	if err == nil {
		ch <- prometheus.MustNewConstMetric(
			deletedMappingsUpDesc,
			prometheus.GaugeValue,
			1.0)
	} else {
		ch <- prometheus.MustNewConstMetric(
			deletedMappingsUpDesc,
			prometheus.GaugeValue,
			0.0)
	}
}

func main() {
	var (
		listenAddress = flag.String("deletedmapping.listen-address", ":9040", "Address to listen on for web interface and telemetry.")
		metricsPath   = flag.String("deletedmapping.telemetry-path", "/metrics", "Path under which to expose metrics.")
		procPath      = flag.String("deletedmapping.proc-path", "/proc", "Path where proc is mounted.")
	)
	flag.Parse()

	exporter, err := NewDeletedMappingExporter(*procPath)
	if err != nil {
		panic(err)
	}
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<html>
			<head><title>Old Lib Exporter</title></head>
			<body>
			<h1>Old Lib Exporter</h1>
			<p><a href='` + *metricsPath + `'>Metrics</a></p>
			</body>
			</html>`))
	})
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
