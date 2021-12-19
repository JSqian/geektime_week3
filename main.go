package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

func main() {
	//使用errgroup
	group, ctx := errgroup.WithContext(context.Background())

	mux := http.NewServeMux()
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, golang!\n")
	})

	// 这个channel负责通知结束
	exitChan := make(chan struct{})
	mux.HandleFunc("/finish", func(w http.ResponseWriter, r *http.Request) {
		exitChan <- struct{}{}
	})

	server := &http.Server{
		Handler: mux,
		//请求监听地址
		Addr: ":8080",
	}

	// server.ListenAndServe()
	// 启动server
	group.Go(func() error {
		return server.ListenAndServe()
	})
	//关闭server
	group.Go(func() error {
		select {
		case <-ctx.Done():
			log.Println("errgroup exit")
			return server.Shutdown(ctx)
		case <-exitChan:
			log.Println("service finish, exit")
			return server.Shutdown(ctx)
		}
	})

	//linux signal信号的注册和处理
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	group.Go(func() error {
		select {
		case <-ctx.Done():
			log.Println("signal goroutine finish")
			return ctx.Err()
		case sig := <-c:
			return errors.Errorf("get os signal: %v", sig)
		}
	})

	if err := group.Wait(); err != nil {
		// if here shows "..., err is get os signal: xxx", means signal goroutine finish
		log.Println("goroutine error, err is", err)
	}
	log.Println("all goroutines finish")
}
