# 第 3 章：free5GC 整體架構與模組介紹

## 3.1 free5GC 是什麼？

free5GC 是一個以 **教學、研究與 PoC（Proof of Concept）** 為主要目標的開源 5G Core 專案。它實作了 3GPP Release 15 為主（現已升級到 Release 17）的 5G 核心網路控制面與使用者面功能，讓開發者可以在一般服務器、虛擬機或容器環境中，快速建立一個可運作的 5G 核心網路實驗平台。

與商用 5G 核心網路產品相比，free5GC 更重視 **可讀性、可修改性與易於部署**。因此，它非常適合：

- 學術或企業內部進行 5G 相關研究與教學
- 驗證新協定、新演算法或新架構的概念
- 搭配開源 RAN、UE 做端到端（E2E）實驗

另一方面，free5GC 並不追求完整覆蓋所有 3GPP 功能，也沒有提供電信等級的 HA / 擴充性設計，因此 **不建議直接用於商用生產環境**。在後續章節中，我們會更多從「理解 5GC 架構與程式設計」的角度，來看待 free5GC，而不是把它當作黑盒子在操作。

目前開源 5GC 專案中，除了 free5GC 之外，還有像其他開源的 5G 核心網路專案。相較之下，free5GC：

- 使用 Go 語言開發，語言本身較容易上手，且具備良好的並行程式設計能力
- 完全遵守 3GPP SBA（Service-Based Architecture），有清楚、模組化與易擴展的實作
- 文件與範例環境偏向教學導向，方便讀者一步步跟著實作

本章的重點，是先為讀者建立 free5GC 的「地圖」，包含整體架構、原始碼倉庫與 NF 程式結構概念，讓你在後面閱讀部署與各 NF 細節時，不會迷路。

> [!Note]
> free5GC 的官方網站：[free5gc.org](https://free5gc.org)

## 3.2 free5GC 的架構介紹

在實務上，free5GC 通常會搭配：

- 一個或多個 UE 端（可以是真實 UE，也可以是模擬 UE）
- 一套 5G RAN（可為真實 gNB，也可以是模擬器）
- 一個 free5GC 5GC 控制面與使用者面服務器

從大方向來看，free5GC 主要實作的是 **5G 核心網路（5GC）**，其內部依照 3GPP 定義拆分為多個 Network Function（NF），並透過 SBA 架構彼此溝通。這些 NF 可以全部部署在同一台機器上，也可以依照實驗需求拆分到不同節點。

在部署型態上，free5GC 常見的使用方式包括：

- **單機部署**：所有 NF 及資料庫併在同一台伺服器上，方便快速體驗與除錯
- **多節點部署**：將資料庫、UPF 或部分 NF 拆出去，模擬較貼近實務的網路拓樸
- **容器化環境**：利用 Docker / Kubernetes 部署，方便與其他系統整合，或在雲端環境進行測試

本節接下來會先說明 free5GC 在 Github 上的主要倉庫，再介紹主倉庫中與網路元件相關的模組，為後面各章節的深入說明鋪路，而詳細的部署方式會在本書的第三部分做介紹。

### 3.2.1 free5GC 在 Github 的主要倉庫

free5GC 在 Github 上並不是只有一個倉庫，而是由[主倉庫](https://github.com/free5gc/free5gc)搭配多個[周邊倉庫（submodule）](https://github.com/free5gc/free5gc/tree/main/NFs)組成一個生態系。

在後續部署章節中，我們會以主倉庫為主，介紹從原始碼取得、建置到啟動 NF 的流程；而本章則聚焦在「從架構與程式結構角度理解主倉庫」。

### 3.2.2 free5GC 主倉庫的網路元件模組

在主倉庫中，free5GC 依照 3GPP 5GC 的定義，將不同網路功能拆分為多個獨立的 Network Function，如下圖所示：

![SBI](./image/sbi.png)

從功能面來看，可以大致分為幾種類型：

- **接入與行動性管理類（Access & Mobility）**

    例如 AMF，負責 UE 的註冊、連線與行動性管理，是 UE 進入 5GC 的主要入口 NF。

- **會話與使用者面管理類（Session & User Plane）**

    例如 SMF 與 UPF，負責 PDU Session 的建立、修改與釋放，以及實際的使用者資料轉送。

- **認證與訂閱資料類（Authentication & Subscription）**

    例如 AUSF、UDM，負責 UE 的認證流程與訂閱資料查詢。

- **策略與計費相關類（Policy & Charging）**

    例如 PCF、CHF，負責提供策略決策與計費數據的反饋。

- **網路切片管理類（Network Slicing）**

    例如 NSSF，協助選擇適當的網路切片與路徑。

在本書後面的章節中，我們會逐一挑選關鍵 NF，從「功能角色 → 訊號流程 → 程式實作」的角度深入解析。為了讓讀者更容易進入程式碼世界，下一節會先談談 free5GC 中單一 NF 的典型程式結構長什麼樣子。

## 3.3 NF 的程式結構

根據目前 free5GC 基於 Release 17 的實作結構，SBI 上的 NF 倉庫內主要結構如下：

```bash
.
├── cmd
│   └── main.go
├── go.mod
├── go.sum
├── internal
│   ├── context
│   ├── logger
│   ├── sbi
│   │   ├── api_xxx.go
│   │   ├── consumer
│   │   ├── processor
│   │   ├── router.go
│   │   └── server.go
│   └── util
├── LICENSE
└── pkg
```

從這個目錄結構，可以大致看出一個 NF 在程式碼層面的組成方式：

- `cmd/main.go`：整個 NF 的啟動入口程式。主要負責讀取設定、建立必要的 context，初始化日誌與各種服務元件，最後啟動 HTTP/SBI server 等對外介面。
- `go.mod` / `go.sum`：Go module 的相依管理檔，記錄這個 NF 使用到的外部套件與版本，確保建置環境一致。
- `internal/context`：通常放與 NF 內部「執行上下文」相關的結構與此 NF 的實際業務邏輯。
- `internal/logger`：封裝日誌相關設定與 API，例如 log level、輸出格式，讓整個 NF 在寫 log 時使用一致的介面。
- `internal/sbi`：與 5GC SBA 上其他 NF 互動的主要實作：

    - `api_xxx.go`：由 OpenAPI / Swagger 產生或手寫的 API 介面定義與 handler 雛形，對應到 3GPP 定義的服務介面。
    - `consumer`：作為「客戶端」角色呼叫其他 NF 的 SBI 服務，例如向 UDM 查詢訂閱資料、向 PCF 取得策略等。
    - `processor`：負責處理 SBI 訊息的商業邏輯，將收到的 request 轉成內部流程，例如更新狀態機、觸發內部上下文模組功能。
    - `router.go` / `server.go`：建立 HTTP router 與實際啟動 SBI server 的程式碼，將 URL path、method 與對應的 handler 串起來。

- `internal/util`：放置這個 NF 會用到的共用小工具，例如字串處理、錯誤包裝、共同常數等，避免散落在各檔案中。
- `pkg`：內部包含此 NF 的一些工廠函數，包含設定檔載入與基本功能的初始化等。
- `LICENSE`：授權條款，說明這個 NF / 專案採用的開源授權模式（free5GC 使用 Apache 2.0），對於後續修改與再利用的合法性非常關鍵。

理解這樣的目錄與模組分工，有助於你在閱讀任何一個 free5GC NF 時，快速找到：

- 從哪裡啟動程式（`cmd/main.go`）
- 從哪裡看內部功能處理邏輯與日誌設定（`context`、`logger`）
- 從哪裡追蹤 SBI 收送流程與行為（`sbi` 底下的 server / consumer / processor）
- 以及哪些工具或共用程式碼可以複用（`util`、`pkg`）

## 3.4 本章小結

本章從三個面向建立了讀者對 free5GC 的「全局地圖」：

- 首先，我們說明了 free5GC 的定位——一個以教學、研究與 PoC 為主的開源 5G Core 平台，強調其優勢在於可讀性與可修改性，而非商用等級的高可用與完整功能。
- 接著，我們以 5GC 架構與 free5GC 在 Github 上的主倉庫為主軸，介紹了各類 NF 的角色與關係，讓讀者能把 AMF、SMF、UPF、PCF 等元件放回 3GPP SBA 的大圖中理解。
- 最後，我們以一個典型 NF 的目錄結構為例，說明從 `cmd/main.go`、`internal` 到 `pkg` 等目錄的職責分工，提供一條實際「讀程式碼」的路線。

<div class="chapter-nav">
  <a href="../chapter4/" class="nav-btn nav-next" title="下一章：AMF 詳解">
    <span class="arrow"></span>
  </a>
</div>
