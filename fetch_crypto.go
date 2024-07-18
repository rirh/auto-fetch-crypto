package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

type ResponseData struct {
	Data     []interface{} `json:"data"`
	Currpage int           `json:"currpage"`
	Maxpage  int           `json:"maxpage"`
}

func getItem(page, size int, wg *sync.WaitGroup, ch chan<- ResponseData, chMaxPage chan<- int) {
	defer wg.Done()
	url := fmt.Sprintf("https://dncapi.bostonteapartyevent.com/api/coin/web-coinrank?page=%d&type=-1&pagesize=%d&webp=1", page, size)
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("accept", "application/json, text/plain, */*")
	req.Header.Set("accept-language", "zh-CN,zh;q=0.9,en;q=0.8,de;q=0.7")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("origin", "https://www.feixiaohao.com")
	req.Header.Set("pragma", "no-cache")
	req.Header.Set("priority", "u=1, i")
	req.Header.Set("referer", "https://www.feixiaohao.com/")
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="125", "Chromium";v="125", "Not.A/Brand";v="24"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"macOS"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "cross-site")
	req.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var responseData ResponseData
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		fmt.Println(err)
		return
	}

	if page == 1 {
		chMaxPage <- responseData.Maxpage
		close(chMaxPage)
	}

	ch <- responseData
}

func main() {
	const size = 100
	var wg sync.WaitGroup
	ch := make(chan ResponseData)
	chMaxPage := make(chan int, 1)

	// Start the first request to get max pages
	wg.Add(1)
	go getItem(1, size, &wg, ch, chMaxPage)

	// Get the max pages from the first request
	maxPages := <-chMaxPage
	fmt.Printf("Max pages: %d\n", maxPages)

	// Start the remaining requests based on max pages
	for page := 2; page <= maxPages; page++ {
		wg.Add(1)
		time.Sleep(time.Millisecond * 200)
		go getItem(page, size, &wg, ch, nil)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var allData []interface{}
	for data := range ch {
		filename := fmt.Sprintf("crypto_%d.json", data.Currpage)
		jsonData, err := json.Marshal(data.Data)
		if err != nil {
			fmt.Println(err)
			return
		}
		err = os.WriteFile(filename, jsonData, 0644)
		if err != nil {
			fmt.Println(err)
			return
		}

		allData = append(allData, data.Data...)
	}

	finalFilename := time.Now().Format("20060102") + "_crypto.json"
	finalJsonData, err := json.Marshal(allData)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = os.WriteFile(finalFilename, finalJsonData, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Remove individual files
	for page := 1; page <= maxPages; page++ {
		filename := fmt.Sprintf("crypto_%d.json", page)
		err = os.Remove(filename)
		if err != nil {
			fmt.Println(err)
		}
	}

	fmt.Printf("Data saved to %s\n", finalFilename)
}
