package main

import (
	"fmt"
	"net/http"
	"github.com/garyburd/redigo/redis"
	"encoding/json"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
	"sort"
	"os"
)

func median(nums []interface{}) float64 {
	var float_prices []float64
	float_prices = make([]float64, 5)
	for i, p := range nums {
		str_price := string(p.([]byte))
		float_price, _ := strconv.ParseFloat(str_price, 64)
		float_prices[i] = float_price
	}
	sort.Float64s(float_prices)
	fmt.Printf("%v\n", float_prices)
	return float_prices[2]
}

func main() {
	// URLs
	trade_url := "https://data.mtgox.com/api/1/BTCUSD/trades"
	post_url := os.ExpandEnv("$ALERT_API_URL")

	// Connect to redis
	r, _ := redis.Dial("tcp", os.ExpandEnv("$REDIS_URL"))

	var tid int64
	tid = 0
	prev_med := 0.0
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

			// Store trade in redis
			r.Do("LPUSH", "mtgox.trades", trade["price"])

			// Update latest tid
			tid, _ = strconv.ParseInt(trade["tid"].(string), 10, 64)
		}
		r.Do("LTRIM", "mtgox.trades", 0, 4)

		// Find median of last 5 trades
		trade_list, _ := r.Do("LRANGE", "mtgox.trades", 0, 4)
		med := median(trade_list.([]interface{}))

		// Post to external service
		if prev_med > 0 {
			buf := "price=" + strconv.FormatFloat(med, 'f', 7, 64) +
				"&timestamp=" + strconv.FormatInt(int64(time.Now().Unix()) * 1000, 10) + 
				"&market=MTGOX"
			fmt.Printf("Posting: %v to %v\n", buf, post_url)
			http.Post(post_url, "application/x-www-form-urlencoded", strings.NewReader(buf))
		}

		// Refresh previous median
		prev_med = med
		time.Sleep(40 * time.Second)
	}

}


