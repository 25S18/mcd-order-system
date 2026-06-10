package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

// グローバルなロガー定義
var AppLog *log.Logger

func initLogger() (*os.File, error) {
	logDir := "logs"
	// logsフォルダが存在しない場合は自動作成
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logPath := filepath.Join(logDir, "order.log")
	// 追記モードでログファイルを開く
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// 標準出力（コンソール）とファイル出力の両方に同時に出力する
	mw := io.MultiWriter(os.Stdout, file)
	AppLog = log.New(mw, "", log.LstdFlags|log.Lmicroseconds)

	return file, nil
}

func main() {
	// ログの初期化
	logFile, err := initLogger()
	if err != nil {
		log.Fatalf("Critical error initializing logger: %v\n", err)
	}
	defer logFile.Close()

	AppLog.Println("[SYSTEM] Starting Order Management Backend Application...")

	// データベース初期化
	if err := InitDB(); err != nil {
		AppLog.Fatalf("[CRITICAL] Database initialization failed: %v\n", err)
	}
	defer CloseDB()

	// Go 1.22+ の新しいマルチプレクサ機能（net/http）を活用したルーティング設定
	mux := http.NewServeMux()

	// 注文管理機能ルーティング
	mux.HandleFunc("OPTIONS /api/orders", HandleCORS)
	mux.HandleFunc("POST /api/orders", HandlePostOrders)
	mux.HandleFunc("GET /api/orders", HandleGetOrders)
	
	mux.HandleFunc("OPTIONS /api/orders/{orderNo}", HandleCORS)
	mux.HandleFunc("GET /api/orders/{orderNo}", HandleGetOrderDetail)
	
	mux.HandleFunc("OPTIONS /api/orders/{orderNo}/status", HandleCORS)
	mux.HandleFunc("PUT /api/orders/{orderNo}/status", HandlePutOrderStatus)

	// フロント掲示板機能ルーティング
	mux.HandleFunc("OPTIONS /api/board", HandleCORS)
	mux.HandleFunc("POST /api/board", HandlePostBoard)

	// 厨房機能ルーティング
	mux.HandleFunc("OPTIONS /api/kitchen", HandleCORS)
	mux.HandleFunc("POST /api/kitchen", HandlePostKitchen)

	// サーバー設定（0.0.0.0:8080 で Listen）
	serverAddr := "0.0.0.0:8080"
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// グレースフルシャットダウンの実装
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		AppLog.Printf("[SYSTEM] Server is listening on %s\n", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			AppLog.Fatalf("[CRITICAL] Server failed to listen: %v\n", err)
		}
	}()

	<-stop
	AppLog.Println("[SYSTEM] Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		AppLog.Fatalf("[CRITICAL] Server forced to shutdown: %v\n", err)
	}

	AppLog.Println("[SYSTEM] Server cleanly stopped.")
}
