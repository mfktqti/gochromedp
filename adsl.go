package main

import (
	"fmt"
	"os/exec"
)

func executeCmd(command string) error {
	cmd := exec.Command("cmd.exe", "/c", command)
	err := cmd.Run()
	return err
}

func connAdsl(adslTitle string, adslName string, adslPass string) {
	adslCmd := "rasdial " + adslTitle + " " + adslName + " " + adslPass
	err := executeCmd(adslCmd)
	if err != nil {
		fmt.Println("ADSL失败:", err)
	}
}

func cutAdsl(adslTitle string) {
	cutCmd := "rasdial " + adslTitle + " /disconnect"
	err := executeCmd(cutCmd)
	if err != nil {
		fmt.Println(adslTitle+" 连接不存在", err)
	}
}
