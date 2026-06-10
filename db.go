package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// DB内部ステータス定数
const (
	StatusReceived  = "オーダ受信済み"
	StatusCooking   = "調理済み"
	StatusDelivered = "受け渡し済み"
)

// InitDB データベース接続設定とテーブルの自動生成を行う
func InitDB() error {
	var err error
	// 同時書き込み・ロック対策のタイムアウト設定を付与
	dsn := "order.db?_busy_timeout=5000"
	db, err = sql.Open("sqlite3", dsn)
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}

	// 同時書き込み対策として最大オープン接続数を1に制限
	db.SetMaxOpenConns(1)

	// 接続確認
	if err = db.Ping(); err != nil {
		return fmt.Errorf("error pinging database: %w", err)
	}

	// テーブルのスキーマ定義と作成
	schema := `
	CREATE TABLE IF NOT EXISTS order_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		order_no TEXT NOT NULL,
		terminal_no TEXT NOT NULL,
		order_status TEXT NOT NULL,
		item_no INTEGER NOT NULL,
		menu_name TEXT NOT NULL,
		unit_price INTEGER NOT NULL,
		quantity INTEGER NOT NULL,
		subtotal INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_order_items_no ON order_items(order_no);
	CREATE INDEX IF NOT EXISTS idx_order_items_status ON order_items(order_status);
	`
	_, err = db.Exec(schema)
	if err != nil {
		return fmt.Errorf("error creating tables: %w", err)
	}

	return nil
}

// CloseDB データベース接続を閉じる
func CloseDB() {
	if db != nil {
		db.Close()
	}
}

// OrderItemEntity DBレコードマッピング用構造体
type OrderItemEntity struct {
	ID          int
	OrderNo     string
	TerminalNo  string
	OrderStatus string
	ItemNo      int
	MenuName    string
	UnitPrice   int
	Quantity    int
	Subtotal    int
	CreatedAt   string
}

// InsertOrderWithNumbering 採番とデータ登録を同一トランザクション内で実行する
func InsertOrderWithNumbering(terminalNo string, items []OrderItemInput) (string, error) {
	tx, err := db.Begin()
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // エラー時は自動ロールバック、コミット後は実質無効化される

	// 現在の日付を取得 (MMDD形式)
	todayStr := time.Now().Format("0102")

	// 本日の最大の注文番号を取得して連番を算出
	var lastOrderNo sql.NullString
	query := `SELECT order_no FROM order_items WHERE order_no LIKE ? ORDER BY order_no DESC LIMIT 1`
	err = tx.QueryRow(query, todayStr+"-%").Scan(&lastOrderNo)
	
	nextSeq := 1
	if err == nil && lastOrderNo.Valid && len(lastOrderNo.String) == 8 {
		// MMDD-NNN 形式から末尾3桁の数値をパース
		var seq int
		_, fmtErr := fmt.Sscanf(lastOrderNo.String[5:], "%03d", &seq)
		if fmtErr == nil {
			nextSeq = seq + 1
		}
	} else if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to scan last order number: %w", err)
	}

	// 新しい注文番号の確定
	newOrderNo := fmt.Sprintf("%s-%03d", todayStr, nextSeq)

	// 明細データのインサート処理
	insertQuery := `
	INSERT INTO order_items (
		order_no, terminal_no, order_status, item_no, menu_name, unit_price, quantity, subtotal
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	stmt, err := tx.Prepare(insertQuery)
	if err != nil {
		return "", fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	for i, item := range items {
		itemNo := i + 1
		subtotal := item.UnitPrice * item.Quantity
		_, err = stmt.Exec(newOrderNo, terminalNo, StatusReceived, itemNo, item.MenuName, item.UnitPrice, item.Quantity, subtotal)
		if err != nil {
			return "", fmt.Errorf("failed to execute insert for item %d: %w", i, err)
		}
	}

	// コミットして確定
	if err = tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return newOrderNo, nil
}

// FetchAllOrders 全注文を取得（条件指定用のステータスは空文字ならフィルタなし）
func FetchAllOrders(statusFilter string) ([]OrderItemEntity, error) {
	var rows *sql.Rows
	var err error

	if statusFilter != "" {
		query := `SELECT id, order_no, terminal_no, order_status, item_no, menu_name, unit_price, quantity, subtotal, created_at 
		          FROM order_items WHERE order_status = ? ORDER BY order_no ASC, item_no ASC`
		rows, err = db.Query(query, statusFilter)
	} else {
		query := `SELECT id, order_no, terminal_no, order_status, item_no, menu_name, unit_price, quantity, subtotal, created_at 
		          FROM order_items ORDER BY order_no ASC, item_no ASC`
		rows, err = db.Query(query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []OrderItemEntity
	for rows.Next() {
		var e OrderItemEntity
		err := rows.Scan(&e.ID, &e.OrderNo, &e.TerminalNo, &e.OrderStatus, &e.ItemNo, &e.MenuName, &e.UnitPrice, &e.Quantity, &e.Subtotal, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, nil
}

// FetchOrderDetails 指定されたorderNoの明細を全件取得する
func FetchOrderDetails(orderNo string) ([]OrderItemEntity, error) {
	query := `SELECT id, order_no, terminal_no, order_status, item_no, menu_name, unit_price, quantity, subtotal, created_at 
	          FROM order_items WHERE order_no = ? ORDER BY item_no ASC`
	rows, err := db.Query(query, orderNo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []OrderItemEntity
	for rows.Next() {
		var e OrderItemEntity
		err := rows.Scan(&e.ID, &e.OrderNo, &e.TerminalNo, &e.OrderStatus, &e.ItemNo, &e.MenuName, &e.UnitPrice, &e.Quantity, &e.Subtotal, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, nil
}

// UpdateOrderStatus 指定した注文番号の全ステータスを更新する
func UpdateOrderStatus(orderNo string, nextStatus string) (int64, error) {
	query := `UPDATE order_items SET order_status = ? WHERE order_no = ?`
	res, err := db.Exec(query, nextStatus, orderNo)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affected, nil
}

// FetchActiveOrderNumbers 掲示板用に指定されたステータスを持つ一意の注文番号リストを抽出する
func FetchActiveOrderNumbers(status string) ([]string, error) {
	query := `SELECT DISTINCT order_no FROM order_items WHERE order_status = ? ORDER BY order_no ASC`
	rows, err := db.Query(query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orderNos []string
	for rows.Next() {
		var no string
		if err := rows.Scan(&no); err != nil {
			return nil, err
		}
		orderNos = append(orderNos, no)
	}
	return orderNos, nil
}