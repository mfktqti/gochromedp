package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/panjf2000/ants"
	"github.com/xuri/excelize/v2"
)

type Para struct {
	Username     string
	Password     string
	Url          string
	AdslUsername string
	AdslPassword string
	CurrentIndex int
	Index        int
}

var WriteResultChan = make(chan string, 1)
var waitGroup sync.WaitGroup

func main() {
	var tempIndex = 1
	if len(os.Args) >= 2 {
		i, err := strconv.Atoi(os.Args[1])
		if err != nil || i <= 0 {
			log.Fatal("传入的启动参数错误")
		}
		tempIndex = i
	}
	defer ants.Release()
	rows, err := readAccount("config.xlsx")
	if err != nil {
		log.Fatalf("读取账号密码出错:%v", err)
	}
	ipList, err := readConfig("iplist.txt")
	if err != nil {
		log.Fatalf("读取Ip列表出错:%v", err)
	}
	adslInfo, err := readConfig("adsl_config.txt")
	if err != nil {
		log.Fatalf("读取Adsl信息出错:%v", err)
	}

	if len(adslInfo) == 2 {
		connAdsl("宽带连接", adslInfo[0], adslInfo[1])
		cutAdsl("宽带连接")
	}

	go WriteResult()

	p, _ := ants.NewPoolWithFunc(1, func(p interface{}) {
		p2 := p.(Para)
		runChromedp(p2)
		waitGroup.Done()
	})
	defer p.Release()
	WriteResultChan <- "开始执行任务..."
	for i := 0; i < len(rows); i++ {
		waitGroup.Add(1)
		url := ""
		if len(ipList) > 0 && i > (len(ipList)-1) {
			url = ipList[i%len(ipList)]
		} else if len(ipList) > 0 {
			url = ipList[i]
		}
		cells := rows[i]
		username := cells[0]
		pass := cells[1]

		if len(url) > 0 && !strings.HasPrefix(url, "http") {
			url = "http://" + url
		}

		para := Para{
			Username:     username,
			Password:     pass,
			Url:          url,
			CurrentIndex: i,
			Index:        tempIndex,
		}
		if len(adslInfo) == 2 {
			para.AdslUsername = adslInfo[0]
			para.AdslPassword = adslInfo[1]
		}
		_ = p.Invoke(para)
	}
	waitGroup.Wait()
	WriteResultChan <- "执行任务完成..."
	close(WriteResultChan)
	time.Sleep(2 * time.Second)
}

func runChromedp(p Para) {
	if p.AdslUsername != "" && p.AdslPassword != "" && p.CurrentIndex%p.Index == 0 {
		connAdsl("宽带连接", p.AdslUsername, p.AdslPassword)

	}
	if p.AdslUsername != "" && p.AdslPassword != "" && (p.CurrentIndex+1)%p.Index == 0 {
		defer cutAdsl("宽带连接")
	}
	var status, points string

	// // create chrome instance
	// ctx, cancel := chromedp.NewContext(
	// 	context.Background(),
	// 	// chromedp.WithDebugf(log.Printf),
	// )
	// defer cancel()

	// // create a timeout
	// ctx, cancel = context.WithTimeout(ctx, 150*time.Second)
	// defer cancel()

	//fmt.Printf("p.URL: %v\n", p.URL)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		//chromedp.ProxyServer("http://218.59.139.238:80"),
		//chromedp.ProxyServer("http://127.0.0.1:41091"),
	)
	if len(p.Url) > 0 {
		opts = append(opts, chromedp.ProxyServer(p.Url))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// create chrome instance
	ctx, cancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
	)
	defer cancel()
	// create a timeout
	ctx, cancel = context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	loginTasks := []chromedp.Action{
		chromedp.Navigate(`https://all.accor.com/usa/index.en.shtml`),
		chromedp.Sleep(2 * time.Second),
		chromedp.DoubleClick(`#onetrust-close-btn-container > button`, chromedp.NodeVisible),
		chromedp.Sleep(2 * time.Second),
		chromedp.WaitVisible(`#link-navigation-primaryHeader > div > div.link-navigation__connectZone > div > div > div > div`),
		chromedp.Click(`#link-navigation-primaryHeader > div > div.link-navigation__connectZone > div > div > div > div`, chromedp.NodeVisible),
		chromedp.Sleep(2 * time.Second),
		chromedp.Click(`#list-items > div:nth-child(5) > div > a`, chromedp.NodeVisible),
		chromedp.Sleep(5 * time.Second),
		chromedp.WaitVisible(`#primary_button`),
		chromedp.SetValue("#username-id", p.Username),
		chromedp.SetValue("#password-id", p.Password),
		chromedp.Sleep(3 * time.Second),
		chromedp.Click(`#primary_button`, chromedp.NodeVisible),
		chromedp.Sleep(5 * time.Second),
	}

	homeTasks := []chromedp.Action{
		chromedp.WaitVisible(`#link-navigation-primaryHeader > div > div.link-navigation__connectZone > div > div > div > div`),
		chromedp.Sleep(5 * time.Second),
		chromedp.Click(`#link-navigation-primaryHeader > div > div.link-navigation__connectZone > div > div > div > div`, chromedp.NodeVisible),
		chromedp.Sleep(1 * time.Second),
	}
	logoutTasks := []chromedp.Action{
		chromedp.Sleep(time.Second),
		chromedp.Click(`#logout-button`, chromedp.NodeVisible),
		chromedp.Sleep(3 * time.Second),
	}

	err := chromedp.Run(ctx, loginTasks...)
	if err != nil {
		fmt.Printf("网络超时: %v\n", err)
		return
	}
	var loginErrorMessage = ""
	for {
		if err = chromedp.Run(ctx, chromedp.Evaluate(`
	var element = document.querySelector("#primary_button")
	element.innerText`, &loginErrorMessage)); err == nil {
			if err = chromedp.Run(ctx, chromedp.Evaluate(`
			var element = document.querySelector("#api-service-error")
			element.innerText`, &loginErrorMessage)); err == nil {
				log.Println("密码错误：" + p.Username)
				return
			}
			chromedp.Run(ctx, chromedp.Sleep(time.Second))
		} else {
			break
		}
	}

	if err := chromedp.Run(ctx, homeTasks...); err != nil {
		fmt.Printf("登录后首页: %v\n", err)
	}

	if err := chromedp.Run(ctx, chromedp.Evaluate(`
	var element = document.querySelector("#list-items > div:nth-child(3) > div > div > a > div.item__wrapper.item__wrapper--text__wrapper > span.value")
	element.innerText`, &status)); err != nil {
		//fmt.Printf("status err: %v\n", err)
		log.Println("没有status的值：" + p.Username)
	}

	if err := chromedp.Run(ctx, chromedp.Evaluate(`
	var element = document.querySelector("#list-items > div:nth-child(4) > div > div > a > div.item__wrapper.item__wrapper--text__wrapper > span.value")
	element.innerText`, &points)); err != nil {
		//fmt.Printf("points err: %v\n", err)
		log.Println("没有points的值：" + p.Username)
	}

	chromedp.Run(ctx, logoutTasks...)
	if status != "" && points != "" {
		content := fmt.Sprintf("\nusername:%s,status:%s,points:%s", p.Username, status, points)
		fmt.Printf("%v\n", content)
		WriteResultChan <- content
	}
}

func WriteResult() {
	logFilePath := "Result_" + time.Now().Format("20060102") + ".txt"
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		_, _ = os.Create(logFilePath)
	}
	// open log file
	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer func() { _ = f.Close() }()

	for v := range WriteResultChan {
		logContent := v
		t := time.Now()
		if _, err = f.WriteString(t.Format(time.ANSIC) + "\r\n" + logContent + "\r\n"); err != nil {
			panic(err)
		}
	}
}

func readConfig(path string) ([]string, error) {
	var results []string
	fileHandler, err := os.OpenFile(path, os.O_RDONLY, 0666)
	if err != nil {
		return results, err
	}
	defer fileHandler.Close()
	reader := bufio.NewReader(fileHandler)

	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		results = append(results, string(line))
	}
	return results, nil
}

func readAccount(path string) ([][]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return [][]string{}, err
	}
	defer func() {
		// Close the spreadsheet.
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	return f.GetRows("Sheet1")
}

// // newProxy creates a proxy that requires authentication.
// func newProxy(url string) *httputil.ReverseProxy {
// 	proxy := func(_ *http.Request) (*neturl.URL, error) {
// 		return neturl.Parse(fmt.Sprintf("http://%s", url))
// 	}

// 	return &httputil.ReverseProxy{
// 		Director: func(r *http.Request) {
// 			if dump, err := httputil.DumpRequest(r, true); err == nil {
// 				log.Printf("%s", dump)
// 			}
// 			// // hardcode username/password "u:p" (base64 encoded: dTpw ) to make it simple
// 			// if auth := r.Header.Get("Proxy-Authorization"); auth != "Basic dTpw" {
// 			// 	r.Header.Set("X-Failed", "407")
// 			// }
// 		},
// 		//Transport: &transport{http.DefaultTransport},
// 		Transport: &transport{&http.Transport{
// 			Proxy: proxy,
// 			// ForceAttemptHTTP2:     true,
// 			// MaxIdleConns:          100,
// 			// IdleConnTimeout:       90 * time.Second,
// 			// TLSHandshakeTimeout:   10 * time.Second,
// 			// ExpectContinueTimeout: 1 * time.Second,
// 		}},
// 		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
// 			if err.Error() == "407" {
// 				log.Println("proxy: not authorized")
// 				w.Header().Add("Proxy-Authenticate", `Basic realm="Proxy Authorization"`)
// 				w.WriteHeader(407)
// 			} else {
// 				w.WriteHeader(http.StatusBadGateway)
// 			}
// 		},
// 	}
// }

// type transport struct {
// 	http.RoundTripper
// }

// func (t *transport) RoundTrip(r *http.Request) (*http.Response, error) {
// 	if h := r.Header.Get("X-Failed"); h != "" {
// 		return nil, fmt.Errorf(h)
// 	}
// 	return t.RoundTripper.RoundTrip(r)
// }
