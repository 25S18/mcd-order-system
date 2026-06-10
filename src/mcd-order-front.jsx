import React, { useState, useEffect } from 'react';

// ==========================================
// 4. モックモード設定 (true: 擬似動作 / false: 実通信)
// ==========================================
const useMock = true;

// ==========================================
// 3.A メニューデータ定義 (マクドナルド風)
// ==========================================
const MENU_DATA = [
  { id: 1, name: "ビッグマック", price: 480, image: "https://example.com/bigmac.jpg", description: "こだわりの100%ビーフパティと特製ソース。" },
  { id: 2, name: "マックフライポテト(M)", price: 380, image: "https://example.com/potato.jpg", description: "外はカリッと、中はホクホクの人気ポテト。" },
  { id: 3, name: "コカ・コーラ(M)", price: 240, image: "https://example.com/cola.jpg", description: "バーガーに最適な定番爽快ドリンク。" },
  { id: 4, name: "チキンマックナゲット 5ピース", price: 260, image: "https://example.com/nugget.jpg", description: "外はカリッとジューシーなチキンナゲット。" },
  { id: 5, name: "てりやきマックバーガー", price: 400, image: "https://example.com/teriyaki.jpg", description: "ポークパティに甘辛いてりやきソース。" },
  { id: 6, name: "チーズバーガー", price: 200, image: "https://example.com/cheeseburger.jpg", description: "まろやかなチーズとビーフパティの調和。" }
];

export default function McdOrderFront() {
  // ==========================================
  // 状態構成 (0:初期設定, 1:メニュー選択, 2:注文確認, 3:注文完了, 4:エラー)
  // ==========================================
  const [screenMode, setScreenMode] = useState(0);
  const [serverUrl, setServerUrl] = useState("");
  const [terminalNo, setTerminalNo] = useState("");
  const [cart, setCart] = useState([]); // [{ menuName, unitPrice, quantity, subtotal }]
  const [currentPage, setCurrentPage] = useState(1);
  const [orderNo, setOrderNo] = useState("");
  const [errorMessage, setErrorMessage] = useState("");
  const [imageErrors, setImageErrors] = useState({}); // 画像読み込みエラー管理

  const itemsPerPage = 3;

  // 画面切り替え時に必ず最上部へスクロール
  const changeScreen = (mode) => {
    setScreenMode(mode);
    window.scrollTo(0, 0);
  };

  // アプリ起動時にlocalStorageから接続情報を復元
  useEffect(() => {
    const savedUrl = localStorage.getItem("serverUrl") || "";
    const savedTerminal = localStorage.getItem("terminalNo") || "";
    setServerUrl(savedUrl);
    setTerminalNo(savedTerminal);
    if (savedUrl && savedTerminal) {
      changeScreen(1);
    }
  }, []);

  // ==========================================
  // 2. 【共通システム要件】URL末尾スラッシュ除去
  // ==========================================
  const sanitizeUrl = (url) => {
    let cleanUrl = url.trim();
    while (cleanUrl.endsWith("/")) {
      cleanUrl = cleanUrl.slice(0, -1);
    }
    return cleanUrl;
  };

  // 設定保存処理
  const handleSaveSettings = (e) => {
    e.preventDefault();
    if (!serverUrl ? true : !terminalNo) {
      alert("すべての項目を入力してください。");
      return;
    }
    const cleanUrl = sanitizeUrl(serverUrl);
    localStorage.setItem("serverUrl", cleanUrl);
    localStorage.setItem("terminalNo", terminalNo.trim());
    setServerUrl(cleanUrl);
    setTerminalNo(terminalNo.trim());
    changeScreen(1);
  };

  // カート操作：数量追加・更新（最大5件、明細内数量1~5、menuName重複禁止）
  const handleAddToCart = (item) => {
    const existingIndex = cart.findIndex(c => c.menuName === item.name);
    
    if (existingIndex >= 0) {
      // 既にカートにある場合、数量更新（1~5の範囲チェック）
      const newQuantity = cart[existingIndex].quantity + 1;
      if (newQuantity > 5) {
        alert("同一商品の数量は最大5個までです。");
        return;
      }
      const updatedCart = [...cart];
      updatedCart[existingIndex].quantity = newQuantity;
      updatedCart[existingIndex].subtotal = item.price * newQuantity;
      setCart(updatedCart);
    } else {
      // 新規追加（明細行数は1〜5件まで）
      if (cart.length >= 5) {
        alert("一度に注文できるメニューは5種類までです。");
        return;
      }
      setCart([...cart, {
        menuName: item.name,
        unitPrice: item.price,
        quantity: 1,
        subtotal: item.price
      }]);
    }
  };

  // カート内の数量直接変更
  const handleUpdateQuantity = (index, delta) => {
    const updatedCart = [...cart];
    const newQty = updatedCart[index].quantity + delta;
    if (newQty < 1 || newQty > 5) return;
    
    updatedCart[index].quantity = newQty;
    updatedCart[index].subtotal = updatedCart[index].unitPrice * newQty;
    setCart(updatedCart);
  };

  // カートから削除
  const handleRemoveFromCart = (index) => {
    const updatedCart = cart.filter((_, i) => i !== index);
    setCart(updatedCart);
  };

  // カート合計金額計算
  const totalAmount = cart.reduce((sum, item) => sum + item.subtotal, 0);

  // ページネーション用計算
  const indexOfLastItem = currentPage * itemsPerPage;
  const indexOfFirstItem = indexOfLastItem - itemsPerPage;
  const currentMenuItems = MENU_DATA.slice(indexOfFirstItem, indexOfLastItem);
  const totalPages = Math.ceil(MENU_DATA.length / itemsPerPage);

  // 画像エラーハンドリング
  const handleImageError = (id) => {
    setImageErrors(prev => ({ ...prev, [id]: true }));
  };

  // ==========================================
  // 注文確定 API送信処理 (POST /api/orders)
  // ==========================================
  const handleConfirmOrder = async () => {
    // フロントエンド側最終入力チェック要件
    if (cart.length < 1 || cart.length > 5) {
      alert("注文明細は1〜5件にしてください。");
      return;
    }
    if (totalAmount < 1) {
      alert("合計金額が不正です。");
      return;
    }

    const requestPayload = {
      messageType: "ORDER_CONFIRM",
      terminalNo: terminalNo,
      totalAmount: totalAmount,
      items: cart
    };

    const targetUrl = `${serverUrl}/api/orders`;

    if (useMock) {
      // モックモード動作：コンソール出力と擬似レスポンス
      console.log("--- [MOCK MODE] API CALL ---");
      console.log(`送信先URL: ${targetUrl}`);
      console.log("HTTPメソッド: POST");
      console.log("送信JSON:\n", JSON.stringify(requestPayload, null, 2));
      
      // 擬似的に注文番号を生成 (MMDD-NNN形式)
      const now = new Date();
      const mmdd = `${String(now.getMonth() + 1).padStart(2, '0')}${String(now.getDate()).padStart(2, '0')}`;
      const dummyOrderNo = `${mmdd}-${String(Math.floor(Math.random() * 900) + 1).padStart(3, '0')}`;
      
      setTimeout(() => {
        setOrderNo(dummyOrderNo);
        setCart([]); // カートをクリア
        changeScreen(3); // 注文完了へ
      }, 800);

    } else {
      // リアル通信モード
      try {
        const response = await fetch(targetUrl, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(requestPayload)
        });

        if (!response.ok) {
          throw new Error(`HTTP Error: ${response.status}`);
        }

        const data = await response.json();
        if (data.result === "OK") {
          setOrderNo(data.orderNo);
          setCart([]);
          changeScreen(3);
        } else {
          setErrorMessage(data.message || "注文登録に失敗しました。");
          changeScreen(4);
        }
      } catch (err) {
        setErrorMessage(`サーバーとの通信に失敗しました: ${err.message}`);
        changeScreen(4);
      }
    }
  };

  // 注文キャンセル・最初に戻る
  const handleCancelOrder = () => {
    setCart([]);
    changeScreen(1);
  };

  // 設定リセット
  const handleResetSettings = () => {
    localStorage.clear();
    setServerUrl("");
    setTerminalNo("");
    setCart([]);
    changeScreen(0);
  };

  // ==========================================
  // UIレンダリング
  // ==========================================
  return (
    <div style={styles.container}>
      {/* ヘッダー */}
      <header style={styles.header}>
        <h1 style={styles.headerTitle}>subasu</h1>
        {screenMode > 0 && (
          <span style={styles.terminalBadge}>端末: {terminalNo}</span>
        )}
      </header>

      {/* (0) 初期設定画面 */}
      {screenMode === 0 && (
        <div style={styles.card}>
          <h2 style={styles.cardTitle}>システム初期設定</h2>
          <form onSubmit={handleSaveSettings}>
            <div style={styles.formGroup}>
              <label style={styles.label}>バックエンド接続先URL:</label>
              <input
                type="text"
                placeholder="http://13.xx.xx.xx:8080"
                value={serverUrl}
                onChange={(e) => setServerUrl(e.target.value)}
                style={styles.input}
              />
            </div>
            <div style={styles.formGroup}>
              <label style={styles.label}>オーダー端末番号:</label>
              <input
                type="text"
                placeholder="例: TERM-01"
                value={terminalNo}
                onChange={(e) => setTerminalNo(e.target.value)}
                style={styles.input}
              />
            </div>
            <button type="submit" style={styles.btnPrimary}>設定を保存して開始</button>
          </form>
        </div>
      )}

      {/* (1) メニュー選択画面 */}
      {screenMode === 1 && (
        <div style={styles.row}>
          {/* 左側：メニューカード一覧 */}
          <div style={styles.mainContent}>
            <h2 style={styles.sectionTitle}>メニューを選んでください</h2>
            <div style={styles.menuGrid}>
              {currentMenuItems.map((item) => (
                <div key={item.id} style={styles.menuCard}>
                  {!imageErrors[item.id] ? (
                    <img
                      src={item.image}
                      alt={item.name}
                      onError={() => handleImageError(item.id)}
                      style={styles.menuImage}
                    />
                  ) : (
                    <div style={styles.noImagePlaceholder}>画像なし</div>
                  )}
                  <div style={styles.menuCardBody}>
                    <h3 style={styles.menuName}>{item.name}</h3>
                    <p style={styles.menuDesc}>{item.description}</p>
                    <div style={styles.menuPriceRow}>
                      <span style={styles.menuPrice}>¥{item.price}</span>
                      <button
                        onClick={() => handleAddToCart(item)}
                        style={styles.btnSuccess}
                      >
                        カートに追加
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>

            {/* ページネーションコントロール */}
            <div style={styles.paginationRow}>
              <button
                disabled={currentPage === 1}
                onClick={() => setCurrentPage(prev => Math.max(prev - 1, 1))}
                style={currentPage === 1 ? styles.btnDisabled : styles.btnSecondary}
              >
                ◀ 前へ
              </button>
              <span style={styles.pageInfo}>{currentPage} / {totalPages} ページ</span>
              <button
                disabled={currentPage === totalPages}
                onClick={() => setCurrentPage(prev => Math.min(prev + 1, totalPages))}
                style={currentPage === totalPages ? styles.btnDisabled : styles.btnSecondary}
              >
                次へ ▶
              </button>
            </div>
          </div>

          {/* 右側：現在のカート状況 */}
          <div style={styles.sidebar}>
            <h2 style={styles.sectionTitle}>現在のカート ({cart.length}/5)</h2>
            {cart.length === 0 ? (
              <p style={styles.emptyText}>カートは空です。</p>
            ) : (
              <div>
                <ul style={styles.cartList}>
                  {cart.map((item, index) => (
                    <li key={index} style={styles.cartItem}>
                      <div style={styles.cartItemInfo}>
                        <span style={styles.cartItemName}>{item.menuName}</span>
                        <span style={styles.cartItemSubtotal}>¥{item.subtotal}</span>
                      </div>
                      <div style={styles.cartItemControls}>
                        <button onClick={() => handleUpdateQuantity(index, -1)} style={styles.btnCircle}>-</button>
                        <span style={styles.cartItemQty}>{item.quantity}</span>
                        <button onClick={() => handleUpdateQuantity(index, 1)} style={styles.btnCircle}>+</button>
                        <button onClick={() => handleRemoveFromCart(index)} style={styles.btnDangerMini}>削除</button>
                      </div>
                    </li>
                  ))}
                </ul>
                <div style={styles.totalRow}>
                  <span>合計金額:</span>
                  <span style={styles.totalPrice}>¥{totalAmount}</span>
                </div>
                <button
                  onClick={() => changeScreen(2)}
                  style={styles.btnPrimaryLarge}
                >
                  注文確認画面へ
                </button>
              </div>
            )}
            <button onClick={handleResetSettings} style={styles.btnLink}>接続先設定を変更する</button>
          </div>
        </div>
      )}

      {/* (2) 注文確認画面 */}
      {screenMode === 2 && (
        <div style={styles.cardLarge}>
          <h2 style={styles.cardTitleCentered}>注文内容のご確認</h2>
          <p style={styles.confirmSubtitle}>以下の内容でキッチンへオーダを送信します。</p>
          
          <table style={styles.table}>
            <thead>
              <tr style={styles.thRow}>
                <th style={styles.th}>メニュー名</th>
                <th style={styles.th}>単価</th>
                <th style={styles.th}>数量</th>
                <th style={styles.th}>小計</th>
              </tr>
            </thead>
            <tbody>
              {cart.map((item, index) => (
                <tr key={index} style={styles.tdRow}>
                  <td style={styles.tdBold}>{item.menuName}</td>
                  <td style={styles.td}>¥{item.unitPrice}</td>
                  <td style={styles.td}>{item.quantity}</td>
                  <td style={styles.tdBold}>¥{item.subtotal}</td>
                </tr>
              ))}
            </tbody>
          </table>

          <div style={styles.totalBox}>
            <span>お支払い総額:</span>
            <span style={styles.totalPriceLarge}>¥{totalAmount}</span>
          </div>

          <div style={styles.btnGroupRow}>
            <button onClick={() => changeScreen(1)} style={styles.btnSecondaryLarge}>修正する（戻る）</button>
            <button onClick={handleConfirmOrder} style={styles.btnPrimaryLargeConfirm}>注文を確定する</button>
          </div>
        </div>
      )}

      {/* (3) 注文完了画面 */}
      {screenMode === 3 && (
        <div style={styles.cardCentered}>
          <div style={styles.successIcon}>✓</div>
          <h2 style={styles.successTitle}>ご注文ありがとうございました！</h2>
          <p style={styles.successText}>ただいま調理を開始いたします。掲示板にお手元の番号が表示されるまでお待ちください。</p>
          
          <div style={styles.orderNoBox}>
            <div style={styles.orderNoLabel}>あなたの呼び出し番号</div>
            <div style={styles.orderNoValue}>{orderNo}</div>
          </div>

          <button onClick={handleCancelOrder} style={styles.btnPrimaryLarge}>トップページへ戻る</button>
        </div>
      )}

      {/* (4) エラー画面 */}
      {screenMode === 4 && (
        <div style={styles.cardCenteredError}>
          <div style={styles.errorIcon}>⚠️</div>
          <h2 style={styles.errorTitle}>エラーが発生しました</h2>
          <p style={styles.errorText}>{errorMessage}</p>
          <div style={styles.btnGroupRow}>
            <button onClick={() => changeScreen(1)} style={styles.btnSecondary}>メニューへ戻る</button>
            <button onClick={() => changeScreen(0)} style={styles.btnPrimary}>設定を確認する</button>
          </div>
        </div>
      )}
    </div>
  );
}

// ==========================================
// 4. インラインスタイリング（マクドナルド風配色）
// ==========================================
const styles = {
  container: {
    fontFamily: '"Helvetica Neue", Arial, "Hiragino Kaku Gothic ProN", Meiryo, sans-serif',
    backgroundColor: '#f8f9fa',
    minHeight: '100vh',
    paddingBottom: '50px',
    color: '#333'
  },
  header: {
    backgroundColor: '#db0007', // マクドナルドレッド
    color: '#fff',
    padding: '15px 30px',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    boxShadow: '0 4px 6px rgba(0,0,0,0.1)'
  },
  headerTitle: {
    fontSize: '32px',
    fontWeight: 'bold',
    margin: 0,
    letterSpacing: '1px'
  },
  terminalBadge: {
    backgroundColor: '#ffbc0d', // マクドナルドゴールド
    color: '#222',
    padding: '8px 16px',
    borderRadius: '20px',
    fontWeight: 'bold',
    fontSize: '16px'
  },
  row: {
    display: 'flex',
    maxWidth: '1200px',
    margin: '30px auto',
    padding: '0 20px',
    gap: '30px'
  },
  mainContent: {
    flex: 2
  },
  sidebar: {
    flex: 1,
    backgroundColor: '#fff',
    borderRadius: '12px',
    padding: '20px',
    boxShadow: '0 4px 12px rgba(0,0,0,0.08)',
    height: 'fit-content'
  },
  sectionTitle: {
    fontSize: '24px',
    borderBottom: '3px solid #ffbc0d',
    paddingBottom: '10px',
    marginBottom: '20px',
    fontWeight: 'bold'
  },
  card: {
    backgroundColor: '#fff',
    maxWidth: '500px',
    margin: '60px auto',
    padding: '40px',
    borderRadius: '16px',
    boxShadow: '0 8px 24px rgba(0,0,0,0.1)',
    textAlign: 'center'
  },
  cardLarge: {
    backgroundColor: '#fff',
    maxWidth: '800px',
    margin: '40px auto',
    padding: '40px',
    borderRadius: '16px',
    boxShadow: '0 8px 24px rgba(0,0,0,0.1)'
  },
  cardCentered: {
    backgroundColor: '#fff',
    maxWidth: '600px',
    margin: '60px auto',
    padding: '50px',
    borderRadius: '16px',
    boxShadow: '0 8px 24px rgba(0,0,0,0.1)',
    textAlign: 'center'
  },
  cardCenteredError: {
    backgroundColor: '#fff',
    maxWidth: '600px',
    margin: '60px auto',
    padding: '40px',
    borderRadius: '16px',
    boxShadow: '0 8px 24px rgba(0,0,0,0.1)',
    textAlign: 'center',
    borderTop: '8px solid #db0007'
  },
  cardTitle: {
    fontSize: '26px',
    marginBottom: '30px',
    fontWeight: 'bold'
  },
  cardTitleCentered: {
    fontSize: '28px',
    textAlign: 'center',
    marginBottom: '10px',
    fontWeight: 'bold'
  },
  confirmSubtitle: {
    textAlign: 'center',
    color: '#666',
    marginBottom: '30px',
    fontSize: '16px'
  },
  formGroup: {
    marginBottom: '25px',
    textAlign: 'left'
  },
  label: {
    display: 'block',
    fontSize: '16px',
    fontWeight: 'bold',
    marginBottom: '8px',
    color: '#555'
  },
  input: {
    width: '100%',
    padding: '14px',
    fontSize: '18px',
    border: '2px solid #ddd',
    borderRadius: '8px',
    boxSizing: 'border-box',
    outline: 'none',
    transition: 'border-color 0.2s',
    ':focus': {
      borderColor: '#ffbc0d'
    }
  },
  menuGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(250px, 1fr))',
    gap: '20px',
    marginBottom: '30px'
  },
  menuCard: {
    backgroundColor: '#fff',
    borderRadius: '12px',
    overflow: 'hidden',
    boxShadow: '0 4px 12px rgba(0,0,0,0.06)',
    display: 'flex',
    flexDirection: 'column'
  },
  menuImage: {
    width: '100%',
    height: '180px',
    objectFit: 'cover'
  },
  noImagePlaceholder: {
    width: '100%',
    height: '180px',
    backgroundColor: '#e9ecef',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    color: '#868e96',
    fontSize: '18px',
    fontWeight: 'bold'
  },
  menuCardBody: {
    padding: '15px',
    display: 'flex',
    flexDirection: 'column',
    flexGrow: 1
  },
  menuName: {
    fontSize: '20px',
    margin: '0 0 8px 0',
    fontWeight: 'bold'
  },
  menuDesc: {
    fontSize: '14px',
    color: '#666',
    margin: '0 0 15px 0',
    flexGrow: 1,
    lineHeight: '1.4'
  },
  menuPriceRow: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center'
  },
  menuPrice: {
    fontSize: '22px',
    fontWeight: 'bold',
    color: '#db0007'
  },
  paginationRow: {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    gap: '20px',
    marginTop: '20px'
  },
  pageInfo: {
    fontSize: '18px',
    fontWeight: 'bold'
  },
  emptyText: {
    color: '#999',
    textAlign: 'center',
    padding: '40px 0',
    fontSize: '16px'
  },
  cartList: {
    listStyle: 'none',
    padding: 0,
    margin: '0 0 20px 0'
  },
  cartItem: {
    padding: '12px 0',
    borderBottom: '1px dashed #eee',
    display: 'flex',
    flexDirection: 'column',
    gap: '8px'
  },
  cartItemInfo: {
    display: 'flex',
    justifyContent: 'space-between',
    fontSize: '16px'
  },
  cartItemName: {
    fontWeight: 'bold',
    maxWidth: '70%',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap'
  },
  cartItemSubtotal: {
    fontWeight: 'bold'
  },
  cartItemControls: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'flex-end',
    gap: '10px'
  },
  cartItemQty: {
    fontSize: '18px',
    fontWeight: 'bold',
    minWidth: '20px',
    textAlign: 'center'
  },
  totalRow: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    borderTop: '2px solid #333',
    paddingTop: '15px',
    marginBottom: '20px',
    fontSize: '18px',
    fontWeight: 'bold'
  },
  totalPrice: {
    fontSize: '26px',
    color: '#db0007',
    fontWeight: 'bold'
  },
  table: {
    width: '100%',
    borderCollapse: 'collapse',
    marginBottom: '30px'
  },
  thRow: {
    backgroundColor: '#f1f3f5',
    borderBottom: '2px solid #dee2e6'
  },
  th: {
    padding: '12px',
    fontSize: '16px',
    textAlign: 'left',
    fontWeight: 'bold'
  },
  tdRow: {
    borderBottom: '1px solid #dee2e6'
  },
  td: {
    padding: '14px 12px',
    fontSize: '16px',
    color: '#495057'
  },
  tdBold: {
    padding: '14px 12px',
    fontSize: '17px',
    fontWeight: 'bold'
  },
  totalBox: {
    display: 'flex',
    justifyContent: 'flex-end',
    alignItems: 'center',
    gap: '20px',
    borderTop: '3px double #db0007',
    paddingTop: '20px',
    marginBottom: '40px',
    fontSize: '20px',
    fontWeight: 'bold'
  },
  totalPriceLarge: {
    fontSize: '36px',
    color: '#db0007',
    fontWeight: 'bold'
  },
  btnGroupRow: {
    display: 'flex',
    justifyContent: 'center',
    gap: '20px'
  },
  successIcon: {
    width: '80px',
    height: '80px',
    backgroundColor: '#28a745',
    color: '#fff',
    borderRadius: '50%',
    fontSize: '48px',
    lineHeight: '80px',
    margin: '0 auto 20px auto',
    fontWeight: 'bold'
  },
  successTitle: {
    fontSize: '28px',
    color: '#28a745',
    marginBottom: '15px',
    fontWeight: 'bold'
  },
  successText: {
    fontSize: '16px',
    color: '#555',
    lineHeight: '1.6',
    marginBottom: '30px'
  },
  orderNoBox: {
    backgroundColor: '#fff9db',
    border: '2px dashed #ffbc0d',
    borderRadius: '12px',
    padding: '20px',
    marginBottom: '40px'
  },
  orderNoLabel: {
    fontSize: '16px',
    color: '#666',
    marginBottom: '5px',
    fontWeight: 'bold'
  },
  orderNoValue: {
    fontSize: '48px',
    color: '#db0007',
    fontWeight: 'bold',
    letterSpacing: '2px'
  },
  errorIcon: {
    fontSize: '64px',
    marginBottom: '15px'
  },
  errorTitle: {
    fontSize: '26px',
    color: '#db0007',
    marginBottom: '15px',
    fontWeight: 'bold'
  },
  errorText: {
    fontSize: '16px',
    color: '#555',
    marginBottom: '30px',
    lineHeight: '1.5'
  },
  // ボタン各種
  btnPrimary: {
    backgroundColor: '#ffbc0d',
    color: '#222',
    border: 'none',
    padding: '12px 24px',
    fontSize: '18px',
    fontWeight: 'bold',
    borderRadius: '8px',
    cursor: 'pointer',
    width: '100%',
    boxShadow: '0 4px 6px rgba(0,0,0,0.1)'
  },
  btnPrimaryLarge: {
    backgroundColor: '#ffbc0d',
    color: '#222',
    border: 'none',
    padding: '15px 30px',
    fontSize: '20px',
    fontWeight: 'bold',
    borderRadius: '8px',
    cursor: 'pointer',
    width: '100%',
    boxShadow: '0 4px 6px rgba(0,0,0,0.1)'
  },
  btnPrimaryLargeConfirm: {
    backgroundColor: '#db0007',
    color: '#fff',
    border: 'none',
    padding: '15px 40px',
    fontSize: '22px',
    fontWeight: 'bold',
    borderRadius: '8px',
    cursor: 'pointer',
    boxShadow: '0 4px 6px rgba(0,0,0,0.1)'
  },
  btnSecondary: {
    backgroundColor: '#e9ecef',
    color: '#495057',
    border: 'none',
    padding: '12px 24px',
    fontSize: '18px',
    fontWeight: 'bold',
    borderRadius: '8px',
    cursor: 'pointer'
  },
  btnSecondaryLarge: {
    backgroundColor: '#e9ecef',
    color: '#495057',
    border: 'none',
    padding: '15px 40px',
    fontSize: '20px',
    fontWeight: 'bold',
    borderRadius: '8px',
    cursor: 'pointer'
  },
  btnSuccess: {
    backgroundColor: '#28a745',
    color: '#fff',
    border: 'none',
    padding: '8px 16px',
    fontSize: '16px',
    fontWeight: 'bold',
    borderRadius: '6px',
    cursor: 'pointer'
  },
  btnCircle: {
    width: '32px',
    height: '32px',
    borderRadius: '50%',
    backgroundColor: '#e9ecef',
    border: 'none',
    fontSize: '18px',
    fontWeight: 'bold',
    cursor: 'pointer',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center'
  },
  btnDangerMini: {
    backgroundColor: '#dc3545',
    color: '#fff',
    border: 'none',
    padding: '4px 8px',
    fontSize: '12px',
    fontWeight: 'bold',
    borderRadius: '4px',
    cursor: 'pointer',
    marginLeft: '5px'
  },
  btnDisabled: {
    backgroundColor: '#e9ecef',
    color: '#adb5bd',
    border: 'none',
    padding: '12px 24px',
    fontSize: '18px',
    fontWeight: 'bold',
    borderRadius: '8px',
    cursor: 'not-allowed'
  },
  btnLink: {
    backgroundColor: 'transparent',
    border: 'none',
    color: '#007bff',
    cursor: 'pointer',
    fontSize: '14px',
    textDecoration: 'underline',
    display: 'block',
    margin: '15px auto 0 auto'
  }
};