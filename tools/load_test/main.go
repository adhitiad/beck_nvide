package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	// 1. Definisikan parameter CLI
	targetHost := flag.String("host", "localhost:8080", "Host target untuk load test")
	connCount := flag.Int("conns", 1000, "Jumlah koneksi WebSocket konkuren yang disimulasikan")
	durationSec := flag.Int("duration", 30, "Durasi simulasi berjalan dalam detik")
	roomID := flag.String("room", "test-lobby-room", "Room ID WebSocket target")
	flag.Parse()

	u := url.URL{Scheme: "ws", Host: *targetHost, Path: "/ws"}
	fmt.Printf("=== NVide WebSocket Load Test Tool (Fase 5) ===\n")
	fmt.Printf("Menargetkan: %s (Room: %s)\n", u.String(), *roomID)
	fmt.Printf("Simulasi: %d Koneksi WebSocket Konkuren selama %d detik\n\n", *connCount, *durationSec)

	// Counters untuk pelacakan performa real-time
	var successCount int64
	var failureCount int64
	var activeConns int64
	var totalLatencyMs int64

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Spawn koneksi secara konkuren dengan sedikit jeda (ramp-up time) agar tidak membanjiri server instan
	rampUpDelay := 5 * time.Millisecond
	for i := 0; i < *connCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			time.Sleep(time.Duration(id) * rampUpDelay)

			start := time.Now()
			// Dial WebSocket
			dialer := websocket.Dialer{
				HandshakeTimeout: 5 * time.Second,
			}

			// Tambahkan parameter query token jika diperlukan (misal bypass auth dalam mode test)
			dialUrl := fmt.Sprintf("%s?room_id=%s&user_id=loadtest-user-%d", u.String(), *roomID, id)
			conn, _, err := dialer.Dial(dialUrl, nil)
			if err != nil {
				atomic.AddInt64(&failureCount, 1)
				return
			}
			defer conn.Close()

			latency := time.Since(start).Milliseconds()
			atomic.AddInt64(&totalLatencyMs, latency)
			atomic.AddInt64(&successCount, 1)
			atomic.AddInt64(&activeConns, 1)
			defer atomic.AddInt64(&activeConns, -1)

			// Reader loop untuk menjaga koneksi tetap hidup dan membaca ping/pong
			readCtx, readCancel := context.WithCancel(ctx)
			defer readCancel()

			go func() {
				for {
					select {
					case <-readCtx.Done():
						return
					default:
						_, _, err := conn.ReadMessage()
						if err != nil {
							return
						}
					}
				}
			}()

			// Tetap terhubung selama durasi load test atau hingga diinterupsi
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					// Graceful close handshake
					_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
					return
				case <-ticker.C:
					// Kirim pesan ping tiruan untuk memicu rate limiting token bucket
					pingMsg := []byte(`{"type":"ping","payload":{}}`)
					if err := conn.WriteMessage(websocket.TextMessage, pingMsg); err != nil {
						return
					}
				}
			}
		}(i)
	}

	// 3. Monitor status load test secara periodik
	stopChan := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		elapsed := 0
		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				elapsed++
				currentActives := atomic.LoadInt64(&activeConns)
				success := atomic.LoadInt64(&successCount)
				failed := atomic.LoadInt64(&failureCount)
				avgLat := int64(0)
				if success > 0 {
					avgLat = atomic.LoadInt64(&totalLatencyMs) / success
				}

				fmt.Printf("[%d d] Aktif: %d | Sukses: %d | Gagal: %d | Rata-rata Latensi Dial: %d ms\n",
					elapsed, currentActives, success, failed, avgLat)

				if elapsed >= *durationSec {
					cancel()
					return
				}
			}
		}
	}()

	// Menangkap sinyal interupsi Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nMenerima sinyal interupsi, menghentikan load test secara gracefully...")
		cancel()
	}()

	wg.Wait()
	close(stopChan)

	// Ringkasan hasil load test
	finalSuccess := atomic.LoadInt64(&successCount)
	finalFailed := atomic.LoadInt64(&failureCount)
	avgLat := int64(0)
	if finalSuccess > 0 {
		avgLat = atomic.LoadInt64(&totalLatencyMs) / finalSuccess
	}

	fmt.Printf("\n=== RINGKASAN LOAD TEST ===\n")
	fmt.Printf("Total Target Koneksi : %d\n", *connCount)
	fmt.Printf("Koneksi Berhasil     : %d\n", finalSuccess)
	fmt.Printf("Koneksi Gagal        : %d\n", finalFailed)
	fmt.Printf("Rata-rata Latensi    : %d ms\n", avgLat)
	fmt.Printf("===========================\n")
}
