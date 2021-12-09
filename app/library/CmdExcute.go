package library

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

func CheckCmdRes(cmdPath string, args []string) (bool, string) {
	log.Println("start excute cmd:", cmdPath, ";args:", args)
	_, excutePathErr := os.Stat(cmdPath)
	if excutePathErr == nil {
		cmd := exec.Command(cmdPath, args...)

		stdout, err := cmd.StdoutPipe()

		if err != nil {
			log.Println("cmd.StdoutPipe...", err)
			return false, ""
		}

		if err := cmd.Start(); err != nil {
			log.Println("cmd.Start...", err)
			return false, ""
		}

		if err != nil {
			log.Println("ioutil.ReadAll...", err)
			return false, ""
		}

		if err := cmd.Wait(); err != nil {
			log.Println("cmd wait...", err)
			return false, ""
		}

		data, err := ioutil.ReadAll(stdout)
		var excuteRes = string(data)
		log.Println("cmd excute result:\n", excuteRes)
		return true, excuteRes
	}
	return false, ""
}
