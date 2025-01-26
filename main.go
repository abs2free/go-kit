package main

import (
	"fmt"

	"github.com/abs2free/go-kit/logger"
	"go.uber.org/zap"
)

func main() {

	// 初始化日志
	log, err := logger.New(zap.DebugLevel)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		return
	}
	defer log.Sync()
	log.Info("this is a test")
	log.Errorf("this is a error message")
}
