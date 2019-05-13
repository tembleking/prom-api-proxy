package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var metricBuffer = &bytes.Buffer{}
var mtx sync.Mutex
var endpoint string

func init() {
	if targetUrl, ok := os.LookupEnv("TARGET_URL"); ok {
		endpoint = targetUrl
	} else {
		panic("TARGET_URL env var not set")
	}
}

func main() {
	http.HandleFunc("/metrics", retrieveMetrics)
	go gatherMetrics()
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func retrieveMetrics(writer http.ResponseWriter, request *http.Request) {
	mtx.Lock()
	defer mtx.Unlock()

	io.Copy(writer, bytes.NewReader(metricBuffer.Bytes()))
}

func gatherFrom(url, token string) (*bytes.Buffer, error) {
	res := &bytes.Buffer{}

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return res, err
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	request = request.WithContext(ctx)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := http.Client{
		Transport: tr,
	}
	response, err := client.Do(request)
	if err != nil {
		return res, err
	}
	defer response.Body.Close()

	_, err = io.Copy(res, response.Body)
	if err != nil {
		return res, err
	}

	return res, nil
}

func gatherMetrics() {
	tokenBytes, err := ioutil.ReadFile("/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		log.Fatal(err)
	}
	token := string(tokenBytes)

	log.Println(fmt.Sprintf("Monitoring %s", endpoint))
	for {
		apiBuffer, err := gatherFrom(endpoint, token)
		if err != nil {
			log.Fatal(err)
		}

		mtx.Lock()
		metricBuffer.Reset()
		io.Copy(metricBuffer, apiBuffer)
		mtx.Unlock()

	}
}
