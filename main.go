package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	neturl "net/url"
	"os"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/xuri/excelize/v2"
)

func main() {

	rows, err := readAccount("config.xlsx")
	if err != nil {
		log.Fatalf("读取账号密码出错:%v", err)
	}
	ipList, err := readIpList("iplist.txt")
	if err != nil {
		log.Fatalf("读取Ip列表出错:%v", err)
	}

	for i := 0; i < len(rows); i++ {
		url := ""
		if i > (len(ipList) - 1) {
			url = ipList[i%len(ipList)]
		} else {
			url = ipList[i]
		}
		cells := rows[i]
		username := cells[0]
		pass := cells[1]

		url = "http://" + url
		fmt.Printf("url: %v\n", url)
		fmt.Printf("username: %v\n", username)
		fmt.Printf("pass: %v\n", pass)
		runChromedp(username, pass, url)
	}
}

func runChromedp(username, password, url string) {
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

	// create a simple proxy that requires authentication
	// p := httptest.NewServer(newProxy(url))
	// defer p.Close()
	//fmt.Printf("p.URL: %v\n", p.URL)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.ProxyServer(url),
		//chromedp.ProxyServer("http://136.228.243.159:8082"),
	//	chromedp.ProxyServer("http://127.0.0.1:41091"),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// create chrome instance
	ctx, cancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
	)
	defer cancel()

	tasks := []chromedp.Action{
		chromedp.Navigate(`https://all.accor.com/usa/index.en.shtml`),
		chromedp.Sleep(2 * time.Second),
		chromedp.DoubleClick(`#onetrust-close-btn-container > button`, chromedp.NodeVisible),
		chromedp.Sleep(2 * time.Second), chromedp.WaitVisible(`#link-navigation-primaryHeader > div > div.link-navigation__connectZone > div > div > div > div`),
		chromedp.Click(`#link-navigation-primaryHeader > div > div.link-navigation__connectZone > div > div > div > div`, chromedp.NodeVisible),
		chromedp.Sleep(2 * time.Second),
		chromedp.Click(`#list-items > div:nth-child(5) > div > a`, chromedp.NodeVisible),
		chromedp.Sleep(5 * time.Second),
		chromedp.WaitVisible(`#primary_button`),
		chromedp.SetValue("#username-id", username),
		chromedp.SetValue("#password-id", password),
		chromedp.Sleep(5 * time.Second),
		chromedp.Click(`#primary_button`, chromedp.NodeVisible),
		chromedp.Sleep(3 * time.Second),
		chromedp.WaitVisible(`#link-navigation-primaryHeader > div > div.link-navigation__connectZone > div > div > div > div`),
		chromedp.Sleep(5 * time.Second),
		chromedp.Click(`#link-navigation-primaryHeader > div > div.link-navigation__connectZone > div > div > div > div`, chromedp.NodeVisible),
		chromedp.Sleep(1 * time.Second),
		chromedp.InnerHTML(`#list-items > div:nth-child(3) > div > div > a > div.item__wrapper.item__wrapper--text__wrapper > span.value`, &status),
		chromedp.InnerHTML(`#list-items > div:nth-child(4) > div > div > a > div.item__wrapper.item__wrapper--text__wrapper > span.value`, &points),
		chromedp.Sleep(time.Second),
		chromedp.Click(`#logout-button`, chromedp.NodeVisible),
		chromedp.Sleep(3 * time.Second),
	}
	// tasks = []chromedp.Action{
	// 	chromedp.Navigate(`https://www.baidu.com`),
	// 	chromedp.Sleep(2 * time.Second),
	// }
	err := chromedp.Run(ctx, tasks...)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	} else {
		content := fmt.Sprintf("\nusername:%s,status:%s,points:%s", username, status, points)
		fmt.Printf("content: %v\n", content)
		WriteResult(content)
	}
}

func WriteResult(logContent string) {
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

	t := time.Now()
	if _, err = f.WriteString(t.Format(time.ANSIC) + logContent + "\n"); err != nil {
		panic(err)
	}
}

func readIpList(path string) ([]string, error) {
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
	f, err := excelize.OpenFile("config.xlsx")
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

// newProxy creates a proxy that requires authentication.
func newProxy(url string) *httputil.ReverseProxy {
	proxy := func(_ *http.Request) (*neturl.URL, error) {
		return neturl.Parse(fmt.Sprintf("http://%s", url))
	}

	return &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			if dump, err := httputil.DumpRequest(r, true); err == nil {
				log.Printf("%s", dump)
			}
			// // hardcode username/password "u:p" (base64 encoded: dTpw ) to make it simple
			// if auth := r.Header.Get("Proxy-Authorization"); auth != "Basic dTpw" {
			// 	r.Header.Set("X-Failed", "407")
			// }
		},
		//Transport: &transport{http.DefaultTransport},
		Transport: &transport{&http.Transport{
			Proxy: proxy,
			// ForceAttemptHTTP2:     true,
			// MaxIdleConns:          100,
			// IdleConnTimeout:       90 * time.Second,
			// TLSHandshakeTimeout:   10 * time.Second,
			// ExpectContinueTimeout: 1 * time.Second,
		}},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if err.Error() == "407" {
				log.Println("proxy: not authorized")
				w.Header().Add("Proxy-Authenticate", `Basic realm="Proxy Authorization"`)
				w.WriteHeader(407)
			} else {
				w.WriteHeader(http.StatusBadGateway)
			}
		},
	}
}

type transport struct {
	http.RoundTripper
}

func (t *transport) RoundTrip(r *http.Request) (*http.Response, error) {
	if h := r.Header.Get("X-Failed"); h != "" {
		return nil, fmt.Errorf(h)
	}
	return t.RoundTripper.RoundTrip(r)
}
