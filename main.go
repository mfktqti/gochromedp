package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/xuri/excelize/v2"
)

func main() {
	// 禁用chrome headless

	f, err := excelize.OpenFile("config.xlsx")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		// Close the spreadsheet.
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		fmt.Println(err)
		return
	}

	for i := 0; i < len(rows); i++ {
		cells := rows[i]
		username := cells[0]
		pass := cells[1]
		runChromedp(username, pass)
	}
}

func runChromedp(username, password string) {
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

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
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