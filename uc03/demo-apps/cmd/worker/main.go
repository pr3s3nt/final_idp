// Worker records a heartbeat in Redis and PostgreSQL every few seconds.
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/pr3s3nt/final_idp/uc03/demo-apps/internal/common"
)

// redisIncr speaks just enough RESP to run INCR.
func redisIncr(addr, key string) (string, error) {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	fmt.Fprintf(conn, "*2\r\n$4\r\nINCR\r\n$%d\r\n%s\r\n", len(key), key)
	return bufio.NewReader(conn).ReadString('\n')
}

func main() {
	common.Start("worker")
	ctx := context.Background()
	redisAddr := net.JoinHostPort(common.Env("REDIS_HOST", true), common.Env("REDIS_PORT", true))
	pool := common.ConnectPostgres(ctx)
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS worker_heartbeats (at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		log.Fatal(err)
	}
	if _, err := redisIncr(redisAddr, "worker:heartbeats"); err != nil {
		log.Fatalf("redis not reachable at %s: %v", redisAddr, err)
	}
	var healthy atomic.Bool
	healthy.Store(true)
	go func() {
		for range time.Tick(5 * time.Second) {
			_, rerr := redisIncr(redisAddr, "worker:heartbeats")
			_, perr := pool.Exec(ctx, `INSERT INTO worker_heartbeats DEFAULT VALUES`)
			healthy.Store(rerr == nil && perr == nil)
		}
	}()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		if !healthy.Load() {
			http.Error(w, "dependency unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	})
	common.Serve("8081", mux)
}
