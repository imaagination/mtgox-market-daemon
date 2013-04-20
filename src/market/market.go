package main

import (
	"fmt"
	"net/http"
	"encoding/json"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
	"sort"
	"container/list"
	"os"
)

func median(nums *list.List) float64 {
	var float_prices []float64
	float_prices = make([]float64, 5)
	e := nums.Front()
	for i := 0; i < 5; i++ {
		float_prices[i] = e.Value.(float64)
		e = e.Next()
	}
	sort.Float64s(float_prices)
	fmt.Printf("%v\n", float_prices)
	return float_prices[2]
}

func main() {
	// URLs
	trade_url := "https://data.mtgox.com/api/1/BTCUSD/trades"
	post_url := os.ExpandEnv("$ALERT_API_URL")

	// Track last fetched trade
	var tid int64
	tid = 0

	// Most recent trades
	tradeHistory := list.New()

	for {
		// Fetch last trades
		var url string
		if tid > 0 { 
			url = trade_url + "?since=" + strconv.FormatInt(tid, 10)
		}	else { 
			url = trade_url 
		}

		resp, _ := http.Get(url)
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		var rawJson interface{}
		json.Unmarshal(body, &rawJson)
		trades := rawJson.(map[string]interface{})
		for _, u := range trades["return"].([]interface{}) {
			trade := u.(map[string]interface{})
			tradePrice := trade["price"].(string)
			float_price, _ := strconv.ParseFloat(tradePrice, 64)

			tradeHistory.PushBack(float_price)

			if tradeHistory.Len() > 5 {
				tradeHistory.Remove(tradeHistory.Front())
			}

			// Update latest tid
			tid, _ = strconv.ParseInt(trade["tid"].(string), 10, 64)
		}

		// Find median of last 5 trades
		med := median(tradeHistory)

		// Post to external service
		buf := "price=" + strconv.FormatFloat(med, 'f', 7, 64) +
			"&timestamp=" + strconv.FormatInt(int64(time.Now().Unix()) * 1000, 10) + 
			"&market=MTGOX"
		fmt.Printf("Posting: %v to %v\n", buf, post_url)
		http.Post(post_url, "application/x-www-form-urlencoded", strings.NewReader(buf))

		// Sleep
		time.Sleep(40 * time.Second)
	}

}


