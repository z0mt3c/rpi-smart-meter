package main

import (
	"log"
	"time"

	"github.com/influxdata/influxdb/client/v2"
)

type InfluxConfig struct {
	URI         string
	Database    string
	Measurement string
	Meter       string
}

var (
	influxClient      client.Client
	influxMeasurement string
	influxTag         map[string]string
	influxBatchConfig = client.BatchPointsConfig{}
)

func setupInflux(config InfluxConfig) {
	c, err := client.NewHTTPClient(client.HTTPConfig{Addr: config.URI})

	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	influxClient = c
	influxMeasurement = config.Measurement
	influxBatchConfig.Database = config.Database
	influxTag = map[string]string{"meter": config.Meter}
}

func writePoints(fields map[string]interface{}) {
	bp, err := client.NewBatchPoints(influxBatchConfig)

	if err != nil {
		log.Fatal(err)
	}

	// Create a point and add to batch
	pt, err := client.NewPoint(influxMeasurement, influxTag, fields, time.Now())

	if err != nil {
		log.Fatal(err)
	}

	bp.AddPoint(pt)

	if err := influxClient.Write(bp); err != nil {
		log.Fatal(err)
	}
}
