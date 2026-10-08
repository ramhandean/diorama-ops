package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

func main() {
	targetURL := flag.String("url", "ws://127.0.0.1:7437/ws", "Target WebSocket URL")
	concurrency := flag.Int("c", 2000, "Number of concurrent connections")
	duration := flag.Duration("d", 15*time.Second, "Test duration")
	flag.Parse()

	fmt.Printf("=== DioramaOps Load Test ===\n")
	fmt.Printf("Target URL:    %s\n", *targetURL)
	fmt.Printf("Concurrency:   %d connections\n", *concurrency)
	fmt.Printf("Duration:      %s\n", *duration)

	var (
		connected   int64
		failed      int64
		messages    int64
		closeErrors int64
	)

	start := time.Now()
	var wg sync.WaitGroup

	// Ramp-up connections in batches to avoid local ephemeral port exhaustion
	batchSize := 100
	rampInterval := 50 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), *duration+10*time.Second)
	defer cancel()

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			sid := fmt.Sprintf("bench_sid_%d", id)
			headers := http.Header{}
			headers.Set("Origin", "http://localhost:3000")

			conn, _, err := websocket.Dial(ctx, *targetURL, &websocket.DialOptions{
				HTTPHeader: headers,
			})
			if err != nil {
				atomic.AddInt64(&failed, 1)
				return
			}
			defer conn.Close(websocket.StatusNormalClosure, "bench complete")

			atomic.AddInt64(&connected, 1)

			// Send hello
			helloMsg := fmt.Sprintf(`{"t":"hello","key":"demo_key","sid":"%s","path":"/bench"}`, sid)
			if err := conn.Write(ctx, websocket.MessageText, []byte(helloMsg)); err != nil {
				atomic.AddInt64(&closeErrors, 1)
				return
			}
			atomic.AddInt64(&messages, 1)

			// Heartbeat ticker
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			timer := time.NewTimer(*duration)
			defer timer.Stop()

			for {
				select {
				case <-timer.C:
					// Send bye
					_ = conn.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf(`{"t":"bye","sid":"%s"}`, sid)))
					return
				case <-ticker.C:
					hbMsg := fmt.Sprintf(`{"t":"hb","sid":"%s","path":"/bench"}`, sid)
					if err := conn.Write(ctx, websocket.MessageText, []byte(hbMsg)); err != nil {
						atomic.AddInt64(&closeErrors, 1)
						return
					}
					atomic.AddInt64(&messages, 1)
				case <-ctx.Done():
					return
				}
			}
		}(i)

		if (i+1)%batchSize == 0 {
			time.Sleep(rampInterval)
		}
	}

	wg.Wait()
	elapsed := time.Since(start)

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	fmt.Println("\n=== Load Test Results ===")
	fmt.Printf("Elapsed:               %.2fs\n", elapsed.Seconds())
	fmt.Printf("Connected Concurrently: %d / %d\n", atomic.LoadInt64(&connected), *concurrency)
	fmt.Printf("Failed Connections:     %d\n", atomic.LoadInt64(&failed))
	fmt.Printf("Total Messages Sent:    %d\n", atomic.LoadInt64(&messages))
	fmt.Printf("Connection Drop/Errors: %d\n", atomic.LoadInt64(&closeErrors))
	fmt.Printf("Client Memory In-Use:   %.2f MB\n", float64(memStats.Alloc)/1024/1024)

	if atomic.LoadInt64(&failed) > int64(*concurrency)/20 {
		fmt.Println("STATUS: FAILED (High connection failures)")
		os.Exit(1)
	}
	fmt.Println("STATUS: PASSED")
}
