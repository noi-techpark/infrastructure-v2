// SPDX-FileCopyrightText: 2024 NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// example-fetcher periodically sends data to the raw-writer-2 ingest service:
//
//   - small: current weather for Bolzano from the Open-Meteo public API (every 60 s)
//   - large: synthetic minute-resolution sensor history (~6 MB JSON, every 5 min)
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"time"
)

// Open-Meteo forecast for Bolzano, South Tyrol — no API key required.
const openMeteoURL = "https://api.open-meteo.com/v1/forecast" +
	"?latitude=46.4983&longitude=11.3548" +
	"&hourly=temperature_2m,precipitation,wind_speed_10m" +
	"&forecast_days=1"

// sensorReading is the schema used for the synthetic large dataset.
type sensorReading struct {
	ID              int     `json:"id"`
	Station         string  `json:"station"`
	Lat             float64 `json:"lat"`
	Lon             float64 `json:"lon"`
	Timestamp       string  `json:"timestamp"`
	TempC           float64 `json:"temp_c"`
	HumidityPct     float64 `json:"humidity_pct"`
	PressureHPa     float64 `json:"pressure_hpa"`
	WindSpeedMS     float64 `json:"wind_speed_ms"`
	WindDirDeg      int     `json:"wind_dir_deg"`
	PrecipMM        float64 `json:"precipitation_mm"`
	RadiationWM2    float64 `json:"solar_radiation_wm2"`
	UVIndex         float64 `json:"uv_index"`
	VisibilityKM    float64 `json:"visibility_km"`
	CloudCoverPct   float64 `json:"cloud_cover_pct"`
	PM25            float64 `json:"pm25_ug_m3"`
	PM10            float64 `json:"pm10_ug_m3"`
	NO2PPB          float64 `json:"no2_ppb"`
	O3PPB           float64 `json:"o3_ppb"`
	BatteryV        float64 `json:"battery_v"`
	SignalStrengthDB int     `json:"signal_strength_db"`
}

func main() {
	writerURL := os.Getenv("WRITER_URL")
	if writerURL == "" {
		writerURL = "http://localhost:8080"
	}

	smallInterval := parseDurationEnv("FETCH_INTERVAL", 60*time.Second)
	largeInterval := parseDurationEnv("LARGE_FETCH_INTERVAL", 5*time.Minute)

	slog.Info("example-fetcher starting",
		"writer", writerURL,
		"small_interval", smallInterval,
		"large_interval", largeInterval,
	)

	client := &http.Client{Timeout: 60 * time.Second}

	// Run both immediately on startup.
	fetchSmall(client, writerURL)
	fetchLarge(client, writerURL)

	smallTicker := time.NewTicker(smallInterval)
	largeTicker := time.NewTicker(largeInterval)
	defer smallTicker.Stop()
	defer largeTicker.Stop()

	for {
		select {
		case <-smallTicker.C:
			fetchSmall(client, writerURL)
		case <-largeTicker.C:
			fetchLarge(client, writerURL)
		}
	}
}

// fetchSmall pulls the current Open-Meteo forecast and posts it as a small message.
func fetchSmall(client *http.Client, writerURL string) {
	body, err := httpGet(client, openMeteoURL)
	if err != nil {
		slog.Error("open-meteo fetch failed", "err", err)
		return
	}
	postToWriter(client, writerURL, "open-meteo", "bolzano-weather", "application/json", body)
}

// fetchLarge generates a synthetic sensor-history dataset (>5 MB) and posts it
// as a large message, triggering the S3 path in raw-writer-2.
func fetchLarge(client *http.Client, writerURL string) {
	body, err := generateSensorHistory(30_000)
	if err != nil {
		slog.Error("generate sensor history failed", "err", err)
		return
	}
	slog.Info("posting large payload", "bytes", len(body))
	postToWriter(client, writerURL, "synthetic", "bolzano-sensor-history", "application/json", body)
}

// generateSensorHistory returns a JSON array of n synthetic minute-resolution
// sensor readings ending at the current time.
func generateSensorHistory(n int) ([]byte, error) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	records := make([]sensorReading, n)
	base := time.Now().UTC().Add(-time.Duration(n) * time.Minute)

	tempC := 12.0
	for i := range records {
		// Random walk so values look plausible.
		tempC += (rng.Float64() - 0.5) * 0.3
		records[i] = sensorReading{
			ID:              i + 1,
			Station:         "Bolzano-Center",
			Lat:             46.4983,
			Lon:             11.3548,
			Timestamp:       base.Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
			TempC:           round2(tempC),
			HumidityPct:     round2(40 + rng.Float64()*50),
			PressureHPa:     round2(1005 + rng.Float64()*20),
			WindSpeedMS:     round2(rng.Float64() * 10),
			WindDirDeg:      rng.Intn(360),
			PrecipMM:        round2(rng.Float64() * 2),
			RadiationWM2:    round2(rng.Float64() * 800),
			UVIndex:         round2(rng.Float64() * 8),
			VisibilityKM:    round2(5 + rng.Float64()*20),
			CloudCoverPct:   round2(rng.Float64() * 100),
			PM25:            round2(rng.Float64() * 25),
			PM10:            round2(rng.Float64() * 50),
			NO2PPB:          round2(rng.Float64() * 40),
			O3PPB:           round2(rng.Float64() * 60),
			BatteryV:        round2(3.5 + rng.Float64()*0.8),
			SignalStrengthDB: -50 - rng.Intn(40),
		}
	}
	return json.Marshal(records)
}

func postToWriter(client *http.Client, writerURL, provider1, provider2, contentType string, body []byte) {
	ts := time.Now().UTC().Format(time.RFC3339)
	url := fmt.Sprintf("%s/%s/%s/%s", writerURL, provider1, provider2, ts)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		slog.Error("build request failed", "provider", provider1+"/"+provider2, "err", err)
		return
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", "example-fetcher")

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("post failed", "provider", provider1+"/"+provider2, "err", err)
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		slog.Error("writer error", "provider", provider1+"/"+provider2, "status", resp.StatusCode, "body", string(respBody))
		return
	}
	slog.Info("ingested", "provider", provider1+"/"+provider2, "bytes", len(body), "status", resp.StatusCode, "response", string(respBody))
}

func httpGet(client *http.Client, url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(os.Getenv(key))
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

func round2(v float64) float64 {
	return float64(int(v*100)) / 100
}
