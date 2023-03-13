package infra

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	checkExecInterval = 5 * time.Second
)

func MonitorExecutable() chan struct{} {
	ch := make(chan struct{})
	go func() {
		exeFilename, _ := os.Executable()
		log.Debug(exeFilename)
		stat, _ := os.Stat(exeFilename)
		originalTime := stat.ModTime()
		for {
			time.Sleep(checkExecInterval)
			stat, _ := os.Stat(exeFilename)
			if !originalTime.Equal(stat.ModTime()) {
				ch <- struct{}{}
				return
			}
		}
	}()
	return ch
}
