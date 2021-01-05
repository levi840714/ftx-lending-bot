package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	_ "github.com/joho/godotenv/autoload"
	"github.com/robfig/cron/v3"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	URL = "https://ftx.com/api"
)

var (
	SUB_ACCOUNT = os.Getenv("SUB_ACCOUNT")
	API_KEY     = os.Getenv("API_KEY")
	SECRET_KEY  = os.Getenv("SECRET_KEY")
	Currency    = os.Getenv("CURRENCY")
)

func init() {
	if SUB_ACCOUNT == "" || API_KEY == "" || SECRET_KEY == "" || Currency == "" {
		log.Fatal("plz set .env file")
	}
	log.Printf("Lending Currency is: %s", Currency)
}

func main() {
	crontab := cron.New()
	crontab.AddFunc("59 * * * *", func() {
		var result string
		timer := time.NewTimer(1 * time.Minute)
	loop:
		for {
			select {
			case <-timer.C:
				break loop
			default:
				balance := GetBalance()
				apy := GetLendingRates()
				result = SubmitLending(apy, balance)
				time.Sleep(5 * time.Second)
			}
		}
		log.Println(result)
	})
	crontab.Start()

	// graceful shutdown
	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	<-shutdown
	log.Printf("lending bot is stopping on %s\n", time.Now().String())
}

type Balance struct {
	Success bool `json:"success"`
	Result  []struct {
		Coin  string  `json:"coin"`
		Free  float64 `json:"free"`
		Total float64 `json:"total"`
	} `json:"result"`
}

type LendingRate struct {
	Result []struct {
		Coin     string  `json:"coin"`
		Estimate float64 `json:"estimate"`
		Previous float64 `json:"previous"`
	} `json:"result"`
	Success bool `json:"success"`
}

type LendingOffer struct {
	Coin string  `json:"coin"`
	Size float64 `json:"size"`
	Rate float64 `json:"rate"`
}

type LendingResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type LendingHistory struct {
	Success bool `json:"success"`
	Result  []struct {
		Coin     string    `json:"coin"`
		Proceeds float64   `json:"proceeds"`
		Rate     float64   `json:"rate"`
		Size     float64   `json:"size"`
		Time     time.Time `json:"time"`
	} `json:"result"`
}

func FtxClient(path string, method string, body []byte) *http.Request {
	params := fmt.Sprintf("%s%s/api%s%s", milliTimestamp(), method, path, string(body))
	h := hmac.New(sha256.New, []byte(SECRET_KEY))
	h.Write([]byte(params))
	sign := hex.EncodeToString(h.Sum(nil))
	req, err := http.NewRequest(method, URL+path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("FTX-KEY", API_KEY)
	req.Header.Set("FTX-SIGN", sign)
	req.Header.Set("FTX-TS", milliTimestamp())
	req.Header.Set("FTX-SUBACCOUNT", SUB_ACCOUNT)
	if err != nil {
		return nil
	}
	return req
}

func GetBalance() (totalBalance float64) {
	client := http.Client{}
	path := "/wallet/balances"
	req := FtxClient(path, "GET", nil)
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Get account balance failed,err: %s", err)
		return
	}
	defer res.Body.Close()
	r, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Get account balance failed,err: %s", err)
		return
	}

	//fmt.Println(string(r))
	var balance Balance
	json.Unmarshal(r, &balance)
	for _, coin := range balance.Result {
		if coin.Coin == Currency {
			totalBalance = coin.Total
		}
	}
	return totalBalance
}

func GetLendingRates() (currencyRate float64) {
	client := http.Client{}
	path := "/spot_margin/lending_rates"
	req := FtxClient(path, "GET", nil)
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Get Lending Rates failed,err: %s", err)
		return
	}
	defer res.Body.Close()
	r, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Get Lending Rates failed,err: %s", err)
		return
	}

	var lend LendingRate
	json.Unmarshal(r, &lend)
	for _, coin := range lend.Result {
		if coin.Coin == Currency {
			currencyRate = coin.Estimate
		}
	}
	return currencyRate
}

func SubmitLending(apy, balance float64) string {
	submitApy := apy * 0.8
	body, _ := json.Marshal(LendingOffer{Coin: Currency, Size: balance, Rate: submitApy})
	client := http.Client{}
	path := "/spot_margin/offers"
	req := FtxClient(path, "POST", body)
	res, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("Submit lending offer failed,err: %s", err)
	}
	defer res.Body.Close()
	r, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Sprintf("Submit lending offer failed,err: %s", err)
	}
	//fmt.Printf("%+v", string(r))

	var lendResp LendingResponse
	json.Unmarshal(r, &lendResp)
	if lendResp.Success == false {
		return fmt.Sprintf("Submit lending offer failed,error: %s", lendResp.Error)
	}

	return fmt.Sprintf("Submit lending offer success, Currency: %s, Size: %f, Lending APY: %f%%, Estimate APY: %f%%,", Currency, balance, submitApy*24*365*100, apy*24*365*100)
}

func milliTimestamp() string {
	return strconv.FormatInt(time.Now().UTC().Unix()*1000, 10)
}
