package main

import (
	"encoding/json"
	"net/http"
)

// --- JSONリクエスト・レスポンス用の構造体定義 ---

type OrderItemInput struct {
	MenuName  string `json:"menuName"`
	UnitPrice int    `json:"unitPrice"`
	Quantity  int    `json:"quantity"` // 👈 ここに「int」を正しく補完しました
	Subtotal  int    `json:"subtotal"`
}

type OrderPostRequest struct {
	MessageType string           `json:"messageType"`
	TerminalNo  string           `json:"terminalNo"`
	TotalAmount int              `json:"totalAmount"`
	Items       []OrderItemInput `json:"items"`
}

type OrderPostResponse struct {
	Result      string `json:"result"`
	OrderNo     string `json:"orderNo,omitempty"`
	OrderStatus string `json:"orderStatus,omitempty"`
	TotalAmount int    `json:"totalAmount,omitempty"`
	Message     string `json:"message,omitempty"`
}

type OrderItemResponse struct {
	ItemNo    int    `json:"itemNo"`
	MenuName  string `json:"menuName"`
	UnitPrice int    `json:"unitPrice"`
	Quantity  int    `json:"quantity"`
	Subtotal  int    `json:"subtotal"`
}

type OrderGroupResponse struct {
	OrderNo     string              `json:"orderNo"`
	TerminalNo  string              `json:"terminalNo"`
	OrderStatus string              `json:"orderStatus"`
	TotalAmount int                 `json:"totalAmount"`
	CreatedAt   string              `json:"createdAt"`
	Items       []OrderItemResponse `json:"items"`
}

type OrderStatusUpdateRequest struct {
	OrderStatus string `json:"orderStatus"`
}

type BoardRequest struct {
	TerminalNo  string `json:"terminalNo"`
	MessageType string `json:"messageType"`
	OrderNo     string `json:"orderNo,omitempty"`
}

type BoardResponse struct {
	Result        string   `json:"result"`
	CookingOrders []string `json:"cookingOrders"`
	ReadyOrders   []string `json:"readyOrders"`
	Message       string   `json:"message,omitempty"`
}

type KitchenRequest struct {
	TerminalNo  string `json:"terminalNo"`
	MessageType string `json:"messageType"`
	OrderNo     string `json:"orderNo,omitempty"`
}

type KitchenOrderItem struct {
	MenuName string `json:"menuName"`
	Quantity int    `json:"quantity"`
}

type KitchenOrderGroup struct {
	OrderNo string             `json:"orderNo"`
	Items   []KitchenOrderItem `json:"items"`
}

type KitchenResponse struct {
	Result  string              `json:"result"`
	Orders  []KitchenOrderGroup `json:"orders"`
	Message string              `json:"message,omitempty"`
}

// --- CORS共通処理用ハンドラ ---

func HandleCORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusNoContent)
}

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
}

// 共通JSONエラー返却関数
func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"result": "NG", "message": message})
}

// --- 注文管理機能ハンドラ群 ---

// POST /api/orders 注文登録
func HandlePostOrders(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	AppLog.Printf("[API IN] POST /api/orders from %s\n", r.RemoteAddr)

	var req OrderPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		AppLog.Printf("[API OUT] 400 Bad Request: JSON parse error: %v\n", err)
		respondWithError(w, http.StatusBadRequest, "Invalid JSON structure")
		return
	}

	// ログに入電文の内容を記録
	reqJSON, _ := json.Marshal(req)
	AppLog.Printf("[DATA IN] Request Body: %s\n", string(reqJSON))

	// 入力チェックバリデーション
	if req.TerminalNo == "" {
		AppLog.Println("[API OUT] 400 Bad Request: terminalNo is missing")
		respondWithError(w, http.StatusBadRequest, "terminalNo is required")
		return
	}
	if req.MessageType != "ORDER_CONFIRM" {
		AppLog.Printf("[API OUT] 400 Bad Request: invalid messageType: %s\n", req.MessageType)
		respondWithError(w, http.StatusBadRequest, "messageType must be 'ORDER_CONFIRM'")
		return
	}
	if req.TotalAmount < 1 {
		AppLog.Println("[API OUT] 400 Bad Request: totalAmount < 1")
		respondWithError(w, http.StatusBadRequest, "totalAmount must be >= 1")
		return
	}
	itemLen := len(req.Items)
	if itemLen < 1 || itemLen > 5 {
		AppLog.Printf("[API OUT] 400 Bad Request: items length %d out of range (1-5)\n", itemLen)
		respondWithError(w, http.StatusBadRequest, "items count must be between 1 and 5")
		return
	}

	menuMap := make(map[string]bool)
	calculatedTotal := 0

	for i, item := range req.Items {
		if item.MenuName == "" {
			AppLog.Printf("[API OUT] 400 Bad Request: item[%d] menuName is empty\n", i)
			respondWithError(w, http.StatusBadRequest, "menuName is required for all items")
			return
		}
		if item.UnitPrice < 1 {
			AppLog.Printf("[API OUT] 400 Bad Request: item[%d] unitPrice < 1\n", i)
			respondWithError(w, http.StatusBadRequest, "unitPrice must be >= 1")
			return
		}
		if item.Quantity < 1 || item.Quantity > 5 {
			AppLog.Printf("[API OUT] 400 Bad Request: item[%d] quantity %d out of range (1-5)\n", i, item.Quantity)
			respondWithError(w, http.StatusBadRequest, "quantity must be between 1 and 5")
			return
		}
		// menuNameの重複禁止バリデーション
		if menuMap[item.MenuName] {
			AppLog.Printf("[API OUT] 400 Bad Request: duplicate menuName detected: %s\n", item.MenuName)
			respondWithError(w, http.StatusBadRequest, "duplicate menuName in items is prohibited")
			return
		}
		menuMap[item.MenuName] = true

		// 小計の自動計算および一致チェック
		subtotal := item.UnitPrice * item.Quantity
		if item.Subtotal != subtotal {
			AppLog.Printf("[API OUT] 400 Bad Request: item[%d] subtotal mismatch. expected %d, got %d\n", i, subtotal, item.Subtotal)
			respondWithError(w, http.StatusBadRequest, "item subtotal mismatch with calculation (unitPrice * quantity)")
			return
		}
		calculatedTotal += subtotal
	}

	// 合計金額の検証
	if calculatedTotal != req.TotalAmount {
		AppLog.Printf("[API OUT] 400 Bad Request: totalAmount mismatch. calculated %d, requested %d\n", calculatedTotal, req.TotalAmount)
		respondWithError(w, http.StatusBadRequest, "totalAmount does not match the sum of item subtotals")
		return
	}

	// トランザクション内での採番とDB登録の実行
	orderNo, err := InsertOrderWithNumbering(req.TerminalNo, req.Items)
	if err != nil {
		AppLog.Printf("[DB ERROR] Failed to register order: %v\n", err)
		respondWithError(w, http.StatusInternalServerError, "Internal Database Registration Error")
		return
	}

	AppLog.Printf("[DB INSERT] Successfully registered order. OrderNo: %s, Items: %d\n", orderNo, itemLen)

	resp := OrderPostResponse{
		Result:      "OK",
		OrderNo:     orderNo,
		OrderStatus: StatusReceived,
		TotalAmount: req.TotalAmount,
		Message:     "Order accepted successfully",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)

	respJSON, _ := json.Marshal(resp)
	AppLog.Printf("[API OUT] 200 OK. Response Body: %s\n", string(respJSON))
}

// GET /api/orders 注文一覧取得（クエリによるステータスフィルタ対応）
func HandleGetOrders(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	statusFilter := r.URL.Query().Get("status")
	AppLog.Printf("[API IN] GET /api/orders (filter status=%s) from %s\n", statusFilter, r.RemoteAddr)

	entities, err := FetchAllOrders(statusFilter)
	if err != nil {
		AppLog.Printf("[DB ERROR] Failed to fetch orders: %v\n", err)
		respondWithError(w, http.StatusInternalServerError, "Database read error")
		return
	}

	// 明細構造を行から注文オブジェクト単位へと集約する処理
	orderMap := make(map[string]*OrderGroupResponse)
	var orderOrder []string

	for _, e := range entities {
		if _, exists := orderMap[e.OrderNo]; !exists {
			orderMap[e.OrderNo] = &OrderGroupResponse{
				OrderNo:     e.OrderNo,
				TerminalNo:  e.TerminalNo,
				OrderStatus: e.OrderStatus,
				TotalAmount: 0,
				CreatedAt:   e.CreatedAt,
				Items:       []OrderItemResponse{},
			}
			orderOrder = append(orderOrder, e.OrderNo)
		}
		
		orderMap[e.OrderNo].Items = append(orderMap[e.OrderNo].Items, OrderItemResponse{
			ItemNo:    e.ItemNo,
			MenuName:  e.MenuName,
			UnitPrice: e.UnitPrice,
			Quantity:  e.Quantity,
			Subtotal:  e.Subtotal,
		})
		orderMap[e.OrderNo].TotalAmount += e.Subtotal
	}

	resultList := make([]OrderGroupResponse, 0)
	for _, no := range orderOrder {
		resultList = append(resultList, *orderMap[no])
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resultList)
	AppLog.Printf("[API OUT] 200 OK. Returned %d aggregated orders\n", len(resultList))
}

// GET /api/orders/{orderNo} 注文詳細取得
func HandleGetOrderDetail(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	orderNo := r.PathValue("orderNo")
	AppLog.Printf("[API IN] GET /api/orders/%s from %s\n", orderNo, r.RemoteAddr)

	if orderNo == "" {
		respondWithError(w, http.StatusBadRequest, "orderNo parameter is required")
		return
	}

	entities, err := FetchOrderDetails(orderNo)
	if err != nil {
		AppLog.Printf("[DB ERROR] Failed to fetch order details for %s: %v\n", orderNo, err)
		respondWithError(w, http.StatusInternalServerError, "Database read error")
		return
	}

	if len(entities) == 0 {
		AppLog.Printf("[API OUT] 404 Not Found: OrderNo %s\n", orderNo)
		respondWithError(w, http.StatusNotFound, "Specified orderNo not found")
		return
	}

	resp := OrderGroupResponse{
		OrderNo:     entities[0].OrderNo,
		TerminalNo:  entities[0].TerminalNo,
		OrderStatus: entities[0].OrderStatus,
		TotalAmount: 0,
		CreatedAt:   entities[0].CreatedAt,
		Items:       []OrderItemResponse{},
	}

	for _, e := range entities {
		resp.Items = append(resp.Items, OrderItemResponse{
			ItemNo:    e.ItemNo,
			MenuName:  e.MenuName,
			UnitPrice: e.UnitPrice,
			Quantity:  e.Quantity,
			Subtotal:  e.Subtotal,
		})
		resp.TotalAmount += e.Subtotal
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
	AppLog.Printf("[API OUT] 200 OK. Details for order %s returned\n", orderNo)
}

// PUT /api/orders/{orderNo}/status 注文状態の直接変更
func HandlePutOrderStatus(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	orderNo := r.PathValue("orderNo")
	AppLog.Printf("[API IN] PUT /api/orders/%s/status from %s\n", orderNo, r.RemoteAddr)

	var req OrderStatusUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON structure")
		return
	}

	status := req.OrderStatus
	if status != StatusReceived && status != StatusCooking && status != StatusDelivered {
		respondWithError(w, http.StatusBadRequest, "Invalid orderStatus transition value")
		return
	}

	affected, err := UpdateOrderStatus(orderNo, status)
	if err != nil {
		AppLog.Printf("[DB ERROR] Status update failed for %s: %v\n", orderNo, err)
		respondWithError(w, http.StatusInternalServerError, "Database update error")
		return
	}

	if affected == 0 {
		respondWithError(w, http.StatusNotFound, "No matching order found to update")
		return
	}

	AppLog.Printf("[DB UPDATE] OrderNo %s status changed to '%s' via direct PUT\n", orderNo, status)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"result": "OK", "orderNo": orderNo, "status": status})
}

// --- フロント掲示板機能ハンドラ ---

func HandlePostBoard(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	AppLog.Printf("[API IN] POST /api/board from %s\n", r.RemoteAddr)

	var req BoardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON structure")
		return
	}

	if req.MessageType != "BOARD_REQUEST" {
		respondWithError(w, http.StatusBadRequest, "messageType must be 'BOARD_REQUEST'")
		return
	}

	if req.OrderNo != "" {
		affected, err := UpdateOrderStatus(req.OrderNo, StatusDelivered)
		if err != nil {
			AppLog.Printf("[DB ERROR] Board update state failed for %s: %v\n", req.OrderNo, err)
			respondWithError(w, http.StatusInternalServerError, "Database state update error")
			return
		}
		if affected > 0 {
			AppLog.Printf("[DB UPDATE] Board Event: OrderNo %s updated to '%s' (受け渡し済み)\n", req.OrderNo, StatusDelivered)
		} else {
			AppLog.Printf("[WARN] Board Event: OrderNo %s update requested but not found\n", req.OrderNo)
		}
	}

	cookingList, err := FetchActiveOrderNumbers(StatusReceived)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database fetch error")
		return
	}

	readyList, err := FetchActiveOrderNumbers(StatusCooking)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database fetch error")
		return
	}

	if cookingList == nil {
		cookingList = []string{}
	}
	if readyList == nil {
		readyList = []string{}
	}

	resp := BoardResponse{
		Result:        "OK",
		CookingOrders: cookingList,
		ReadyOrders:   readyList,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)

	respJSON, _ := json.Marshal(resp)
	AppLog.Printf("[API OUT] POST /api/board Response: %s\n", string(respJSON))
}

// --- 厨房機能ハンドラ ---

func HandlePostKitchen(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	AppLog.Printf("[API IN] POST /api/kitchen from %s\n", r.RemoteAddr)

	var req KitchenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON structure")
		return
	}

	if req.MessageType != "KITCHEN_REQUEST" {
		respondWithError(w, http.StatusBadRequest, "messageType must be 'KITCHEN_REQUEST'")
		return
	}

	if req.OrderNo != "" {
		affected, err := UpdateOrderStatus(req.OrderNo, StatusCooking)
		if err != nil {
			AppLog.Printf("[DB ERROR] Kitchen update state failed for %s: %v\n", req.OrderNo, err)
			respondWithError(w, http.StatusInternalServerError, "Database state update error")
			return
		}
		if affected > 0 {
			AppLog.Printf("[DB UPDATE] Kitchen Event: OrderNo %s updated to '%s' (調理済み)\n", req.OrderNo, StatusCooking)
		} else {
			AppLog.Printf("[WARN] Kitchen Event: OrderNo %s update requested but not found\n", req.OrderNo)
		}
	}

	entities, err := FetchAllOrders(StatusReceived)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database fetch error")
		return
	}

	kitchenGroupMap := make(map[string]*KitchenOrderGroup)
	var orderOrder []string

	for _, e := range entities {
		if _, exists := kitchenGroupMap[e.OrderNo]; !exists {
			kitchenGroupMap[e.OrderNo] = &KitchenOrderGroup{
				OrderNo: e.OrderNo,
				Items:   []KitchenOrderItem{},
			}
			orderOrder = append(orderOrder, e.OrderNo)
		}
		kitchenGroupMap[e.OrderNo].Items = append(kitchenGroupMap[e.OrderNo].Items, KitchenOrderItem{
			MenuName: e.MenuName,
			Quantity: e.Quantity,
		})
	}

	ordersResponse := make([]KitchenOrderGroup, 0)
	for _, no := range orderOrder {
		ordersResponse = append(ordersResponse, *kitchenGroupMap[no])
	}

	resp := KitchenResponse{
		Result: "OK",
		Orders: ordersResponse,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)

	respJSON, _ := json.Marshal(resp)
	AppLog.Printf("[API OUT] POST /api/kitchen Response: %s\n", string(respJSON))
}