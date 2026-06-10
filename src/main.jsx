import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.jsx'

// 外部CSS（index.cssなど）をインポートしない構成
ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)