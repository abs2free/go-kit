package monitor

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func MonitorByPromethues(addr string, log *zap.SugaredLogger) {
	// Create non-global registry.
	reg := prometheus.NewRegistry()

	// Add go runtime metrics and process collectors.
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	// Expose /metrics HTTP endpoint using the created custom registry.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Fatal(http.ListenAndServe(addr, nil))
}

func Monitor(cancel context.CancelFunc, addr string, log *zap.SugaredLogger) {
	// 监测
	go func() {
		ticker := time.NewTicker(time.Second * 5)
		var mem runtime.MemStats

		for {
			<-ticker.C
			log.Infof("goroutine 数量: %d \n", runtime.NumGoroutine())
			runtime.ReadMemStats(&mem)
			log.Infof("Alloc = %v kB\n", mem.Alloc/1024/8)
		}
	}()

	// 性能分析
	go func() {
		log.Infoln("pprof start:", addr)
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			log.Infof("listen has a err:%v", err)
			cancel()
		}
	}()
}

func signalCheck(cancel context.CancelFunc) {
	// 信号量监控
	sg := make(chan os.Signal, 1)
	// Trigger graceful shutdown on SIGINT or SIGTERM.
	// The default signal sent by the `kill` command is SIGTERM,
	// which is taken as the graceful shutdown signal for many systems, eg., Kubernetes, Gunicorn.
	signal.Notify(sg, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sg
		fmt.Printf("%s received.\n", sig.String())
		cancel()
	}()
}
