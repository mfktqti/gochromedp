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
	"time"

	"github.com/chromedp/chromedp"
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

func main() {
	var tempIndex = 1
	if len(os.Args) >= 2 {
		i, err := strconv.Atoi(os.Args[1])
		if err != nil || i <= 0 {
			log.Fatal("传入的启动参数错误")
		}
		tempIndex = i
	}

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

	WriteResultChan <- "开始执行任务..."
	for i := 0; i < len(rows); i++ {
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
		runChromedp(para)
	}

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
	log.Printf("打开浏览器: \n")
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.DisableGPU,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("ignore-certificate-errors", true),
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
	ctx, cancel = context.WithTimeout(ctx, 150*time.Second)
	defer cancel()
	log.Printf("打开浏览器完成: \n")
	ariaExpandedStr := ""
	isOk := false

	err := chromedp.Run(ctx,
		chromedp.Navigate(`https://all.accor.com/usa/index.en.shtml`),
		chromedp.DoubleClick(`#onetrust-close-btn-container > button`, chromedp.NodeVisible),
	)
	if err != nil {
		log.Printf("打开首页: %v\n", err)
		return
	}
	log.Printf("打开首页: \n")
	for {
		err = chromedp.Run(ctx,
			chromedp.WaitVisible(`#link-navigation-primaryHeader > div > div.link-navigation__connectZone > div > div > div > div`),
			chromedp.Click(`#link-navigation-primaryHeader > div > div.link-navigation__connectZone > div > div > div > div`, chromedp.NodeVisible),
			chromedp.AttributeValue(`#link-navigation-primaryHeader > div > div.link-navigation__connectZone > div > div > div > div.button-logo > button`, "aria-expanded", &ariaExpandedStr, &isOk),
		)
		if err != nil {
			log.Printf("login菜单 : %v\n", err)
			return
		}
		if isOk && ariaExpandedStr == "true" {
			break
		}
		chromedp.Run(ctx, chromedp.Sleep(time.Second))
	}
	log.Printf("打开登录页: \n")
	//点击登录菜单
	err = chromedp.Run(ctx, chromedp.Click(`#list-items > div:nth-child(5) > div > a`, chromedp.NodeVisible))
	if err != nil {
		log.Printf("点击登录菜单: %v\n", err)
		return
	}
	//点击登录
	err = chromedp.Run(ctx,
		chromedp.WaitVisible(`#primary_button`),
		chromedp.SetValue("#username-id", p.Username),
		chromedp.SetValue("#password-id", p.Password),
		chromedp.Click(`#primary_button`, chromedp.NodeVisible),
		chromedp.WaitNotPresent(`#primary_button`),
	)
	if err != nil {
		log.Printf("点击登录: %v\n", err)
		return
	}
	var title = ""
	//点击登录
	err = chromedp.Run(ctx,
		chromedp.Title(&title),
	)
	if err != nil {
		log.Printf("获取title: %v\n", err)
		return
	}
	if title == "Log in" {
		log.Println("密码错误：" + p.Username)
		return
	}
	log.Printf("跳转到首页: \n")
	for {
		err = chromedp.Run(ctx,
			chromedp.WaitVisible(`#link-navigation-primaryHeader > div > div.link-navigation__connectZone > div > div > div > div`),
			chromedp.Click(`#link-navigation-primaryHeader > div > div.link-navigation__connectZone > div > div > div > div`, chromedp.NodeVisible),
			chromedp.AttributeValue(`#link-navigation-primaryHeader > div > div.link-navigation__connectZone > div > div > div > div.button-logo > button`, "aria-expanded", &ariaExpandedStr, &isOk))
		if err != nil {
			log.Printf("HOME菜单 : %v\n", err)
			return
		}
		if isOk && ariaExpandedStr == "true" {
			break
		}
		chromedp.Run(ctx, chromedp.Sleep(time.Second))
	}
	log.Printf("加载首页完成: \n")
	if err := chromedp.Run(ctx, chromedp.Evaluate(`
	var element = document.querySelector("#list-items > div:nth-child(3) > div > div > a > div.item__wrapper.item__wrapper--text__wrapper > span.value")
	element.innerText`, &status)); err != nil {
		//fmt.Printf("status err: %v\n", err)
		log.Println("没有status的值：" + p.Username)
		return
	}

	if err := chromedp.Run(ctx, chromedp.Evaluate(`
	var element = document.querySelector("#list-items > div:nth-child(4) > div > div > a > div.item__wrapper.item__wrapper--text__wrapper > span.value")
	element.innerText`, &points)); err != nil {
		//fmt.Printf("points err: %v\n", err)
		log.Println("没有points的值：" + p.Username)
	}

	chromedp.Run(ctx, chromedp.Click(`#logout-button`, chromedp.NodeVisible))

	if status != "" && points != "" {
		content := fmt.Sprintf("username:%s,status:%s,points:%s", p.Username, status, points)
		log.Printf("%v\n", content)
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
