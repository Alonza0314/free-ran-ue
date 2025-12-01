# 第 5 章：SMF、UPF 與使用者面流程

![sbi](./image/sbi.png)

在 5G 核心網路中，**SMF（Session Management Function）**與 **UPF（User Plane Function）**是一組「一文一武」的搭檔：SMF 負責幫 UE 設計與管理每一條 PDU 會話的「路線與規則」，UPF 則是在資料面實際收送與轉發封包的節點。當 UE 完成註冊後，SMF 會協同 PCF 等 NF，依據訂閱資料與策略建立或修改 PDU 會話，並透過 N4 介面的 PFCP 把對應的資料面規則下發到一個或多個 UPF，讓整個網路得以正確地轉送 UE 的上下行流量。本章會先從 SMF、UPF 的角色與互動關係談起，再在後續小節中逐步拆解具體的使用者面流程與程式實作細節。

## 5.1 N3/N4 介面與相關協定

### 5.1.1 N3

N3 介面是 **RAN（gNB）與 UPF 之間的使用者面介面**，主要承載 UE 的真正數據封包（例如瀏覽網頁、看影片、企業 VPN 流量等）。從拓樸來看，它連接的是「無線網路的出口」和「核心網使用者面的入口」：

UE 的 IP 封包先在 gNB 被封裝成 GTP-U 封包，沿著 N3 傳到 UPF，之後再由 UPF 依照 PFCP 規則解封、轉送到其他 UPF 或外部資料網路。

值得注意的是，N3 本身只是一條「邏輯介面」的名稱，實際上在協定堆疊上會跑 GTP-U（使用 UDP/IP 承載）。同樣的技術在 4G 中也被用在 eNodeB 與 S-GW 之間的 S1-U 介面，因此如果你對 4G 已經熟悉，可以把 N3 理解成「5G 版本的 S1-U」，只是對接節點從 S-GW 換成了 UPF。

### 5.1.2 GTP

GTP（GPRS Tunneling Protocol）是一系列「為行動網路設計的封包封裝與控制協定」，在 5G 使用者面最常見的是 **GTP-U（User Plane）**。它的核心概念很簡單：

在原本的 IP 封包外面再包一層 GTP-U header，利用 TEID（Tunnel Endpoint Identifier）來標識「這個封包屬於哪一條隧道／哪一條 PDU 會話」。

> [!Note]
> 所謂的 GTP-U 隧道其實就是在 RAN 與 UPF 之間協定好的一組封包標頭表示資訊看！

在 N3 上，gNB 會：

- 根據 UE 的 RAN 內部上下文，挑選對應的 TEID 與 UPF 目的 IP；
- 把 UE 的 IP 封包包成 GTP-U，送往 UPF。

而 UPF 收到封包後，會：

- 先根據外層的 TEID 查表找到對應的 PDR（這一步決定「這是哪一條 PDU 會話」）；
- 再依照 PDR/FAR 的設定，決定是否要解封、往哪裡送出、是否要重新封裝成另一條 GTP-U 隧道。

從除錯角度來看，理解 GTP-U 的 header 與 TEID 概念，會對你分析 Wireshark trace、確認某條流量為什麼走到特定 UPF 介面非常有幫助。

### 5.1.3 N4

N4 介面是 **SMF 與 UPF 之間的控制介面**，主要用來下發、更新與刪除資料面轉送規則。對 SMF 來說，N4 就像是一條「遙控通道」，透過它可以告訴 UPF：哪些封包屬於哪個 PDU 會話、該往哪裡轉送、要套用什麼樣的 QoS 與計費策略。

### 5.1.4 PFCP

在 N4 介面上使用的協定是 PFCP（Packet Forwarding Control Protocol）。PFCP 是一種控制平面的訊號協定，負責在 SMF 與 UPF 之間：

- 建立、修改、刪除 **會話（Session）**，對應到一條 PDU 會話在 UPF 上的資料面狀態
- 下發與更新 **PDR/FAR/QER 等轉送與 QoS 規則**（本書於 5.3.3 會再詳細介紹各欄位與行為）
- 回報計費相關的使用量統計與事件

> [!Tip]
> 在 high level 上，你可以先把 PFCP 想像成：「SMF 告訴 UPF 要怎麼處理封包」的語言，至於每一種規則的欄位、編碼與實作細節，我們會在本章後面配合程式碼一併說明。

從流程角度來看，一條 PDU 會話在 N4/PFCP 上大致會經歷：

1. **建立階段**：SMF 選好要使用的 UPF 後，先透過 PFCP Session Establishment 建立一條對應的 Session，並在其中附上第一批 PDR/FAR/QER 規則，讓 UPF 知道某些 GTP-U 封包或內部流量應該如何分類與轉送。
2. **更新階段**：當 UE 的服務需求或網路狀態改變時（例如 QoS 調整、換路徑、加/減一台 UPF），SMF 會發送 PFCP Session Modification 更新既有規則。
3. **釋放階段**：在 UE 註銷或 PDU 會話結束時，SMF 會用 PFCP Session Deletion 告訴 UPF 可以清除對應 Session 與規則，回收資源並停止統計。

實務上，一條看似簡單的「上網連線」，在 PFCP 裡面可能分成多條規則：例如普通數據、VoNR 語音、特定企業專網流量會對應不同的 PDR/QER，使得之後你在追 code 或看 log 時，可以精準對應每個封包為何會被這樣處理。

## 5.2 SMF（Session Management Function）

### 5.2.1 SMF 的主要功能

SMF 是 5GC 中負責 **PDU 會話管理** 的核心控制面 NF。它根據 UE 的請求（NAS-SM）、訂閱資料（UDM）、策略（PCF）以及網路拓樸（可用的 UPF 與路徑），決定：

- 是否允許建立或修改一條 PDU 會話
- 要選哪一個（或多個）UPF、走哪一條路徑
- 該會話應該套用哪些 **QoS**、**計費**與**切片**等策略（這些策略透過 SBI 向 PCF 發起服務請求來獲取）

在 free5GC 中，SMF 會同時面對兩個世界：

- 一邊透過 SBI 與 AMF、PCF、UDM 等 NF 溝通，負責接收來自 UE 的 PDU Session 請求（NAS-SM 經由 AMF 轉入）、取得訂閱與策略資訊。
- 另一邊透過 N4/PFCP 控制 UPF，把「決定好要怎麼走」的路線與規則，轉換成具體的 PDR/FAR/QER 等資料面配置。

> [!Tip]
> 如果用一個生活化的比喻，你可以把 SMF 想像成「高速公路的路網規劃中心」：
>
> - AMF 帶著「某一台車（UE）想去哪裡（DNN/S-NSSAI）」來詢問
> - SMF 根據車子的身份（訂閱）、票價方案（策略）、當前路況（拓樸與負載），決定要走哪幾條高速公路、在哪裡進出匝道
> - 然後透過 PFCP 告訴沿途的收費站／匝道控制器（UPF）如何認得這台車、該放行到哪裡

### 5.2.2 PDU 會話

在 5G 核心網路中，一個 **PDU 會話（PDU Session）** 可以被想像成 UE 與某一個資料網路（DN，例如 Internet 或企業專網）之間的一條**邏輯專線**。只要這條會話存在，UE 就可以透過它把封包送進 5GC，由 SMF / UPF 依照規則轉送到正確的目的地；當會話被釋放時，相關的 IP 位址、QoS 配置與資料面路徑也會一併被回收。

從組成來看，一條 PDU 會話通常會包含幾個關鍵元素：

- **PDN/DNN**：這條會話要連到哪一個資料網路（例如 `internet`、企業 APN、專網 DNN 等）
- **S-NSSAI / Slice**：這條會話屬於哪一個網路切片（對應不同客戶或業務場景）
- **IP 位址或其他 PDU Type**：例如 IPv4、IPv6、IPv4v6、Ethernet、Unstructured 等（目前 free5GC 官方僅支援 IPv4 版本）
- **QoS Profile / QoS Flow**：會話內可以再細分成一個或多個 QoS Flow（每個對應不同 QFI），用來實現更細緻的服務品質控制

與 4G EPC 時代相比，PDU 會話大致上扮演類似 EPS Bearer 的角色，但 5G 在設計上把：

- 「一條會話」與「多個 QoS Flow」分開處理，讓同一條 PDU 會話可以承載不同 QoS 要求的流量（例如一般數據與語音可以共用會話，但使用不同 QFI）
- 「PDU 會話」與「切片 / DNN」更緊密地綁在一起，方便依據業務需求建立不同類型的連線（例如同一支 UE 可同時對公網與企業專網各有一條 PDU 會話）

## 5.3 UPF（User Plane Function）

### 5.3.1 UPF 的主要功能

UPF 則是 5GC 中的 **資料面工作元件（data-plane worker）**。它負責根據 SMF 下發的 PFCP 規則，對經過的封包做：

- 分類（matching）：判斷這個封包屬於哪一條 PDU 會話、哪一個 QFI / QoS 流
- 轉送（forwarding）：轉到 RAN、其他 UPF，或外部 DN（例如 Internet、專網）
- 處理（processing）：例如量測與計費統計、封包複寫鏡像、部分 header 修改等

從實作角度看，UPF 更貼近一般我們熟悉的高效封包轉送引擎（如 DPDK、eBPF、P4 等概念），但在本書中，我們會著重在「它如何理解 SMF 下發的 PFCP 規則，並把它套用在封包路徑上」，而不會深入每一種硬體或軟體加速技術的細節。

以 free5GC 的實作為例，一個 UPF 實例通常會具備：

- 與 RAN 之間的 N3 介面（GTP-U），負責收送來自 gNB 的使用者面封包
- 與外部資料網路（DN，例如 Internet、企業專網）的介面，用來把封包送出或接回
- 與其他 UPF 之間的 N9 介面，用於分層或多節點 UPF 架構（例如 UL-CL、Branching UPF 等設計）

UPF 在收到封包時的基本流程可以簡化為：

1. 先根據 TEID、5-tuple 或其他欄位，查表找到對應的 PDR；
2. 根據 PDR 指定的 FAR，決定要往哪個介面或下一個節點送出，以及是否需要做 encapsulation / decapsulation；
3. 套用 QER/QFI 等 QoS 設定，並依照需要進行排程、整形或優先權處理；
4. 更新用量統計與事件資訊，以便之後透過 PFCP 回報給 SMF/CHF 作為計費或監測依據。

### 5.3.2 GTP5G

在實作層面，free5GC 的 UPF 並不是「自己從零開始實作整套 GTP-U 協定與封包轉送」，而是大量倚賴 free5GC 自行開發的 Linux 核心模組 **gtp5g** 來處理底層的封包封裝與轉送工作。可以把 gtp5g 想像成「一個專門給 5G 使用的 GTP-U 處理引擎」，被整合在 Linux 核心裡，提供給 UPF 程式呼叫。

gtp5g 主要負責幾件事情：

- 在 Linux 核心中建立與管理 GTP-U「隧道」與對應的 TEID／IP 映射關係
- 根據 user space（UPF）設定的規則，對進出特定網卡的封包做 GTP-U 封裝與解封
- 與 Linux routing / qdisc / nftables 等機制配合，實際完成封包轉送與部分 QoS／過濾行為

而 free5GC UPF 本身，則透過 netlink / ioctl 等機制與 gtp5g 溝通：

- 當從 SMF 收到新的 PFCP 規則時，UPF 會把對應的 TEID、遠端 IP、比對條件等資訊轉寫到 gtp5g
- 當流量經過時，實際做 encapsulation / decapsulation 與路由查表的是 Linux 核心裡的 gtp5g

這樣的設計有兩個好處：

- 使用者面資料平面可以充分利用 Linux 核心既有的封包處理能力與優化（例如多核心、硬體 offload）
- UPF 程式可以把心力放在 PFCP 規則管理與 5GC 邏輯上，而不是重造一個高效能封包轉送引擎

### 5.3.3 資料平面的規則

| 規則 | 全名 | 描述 |
| - | - | - |
| PDR | Packet Detection Rule | 「封包偵測規則」，負責定義**哪些封包屬於哪一條 PDU 會話／哪一個流量**。常見的比對條件包含 TEID、UE IP、5-tuple、QFI 等，是整個資料平面規則鏈的入口。 |
| FAR | Forwarding Action Rule | 「轉送動作規則」，描述當封包被某條 PDR 命中時，**接下來要怎麼處理**：例如要不要轉送、往哪個介面或遠端 IP/TEID 送出、是否要丟棄或作本地端處理等。 |
| QER | QoS Enforcement Rule | 「QoS 執行規則」，用來**套用頻寬、排程與優先權等 QoS 設定**，並與 QFI / QoS Flow 對應。透過 QER，UPF 可以對不同流量給予不同的速率限制、排程權重或保障。 |
| URR | Usage Reporting Rule | 「使用量回報規則」，負責**統計流量與事件並回報給 SMF/CHF**，例如計算上下載位元數、封包數，或在特定條件（如流量超過門檻、會話結束）時觸發回報，支援計費與監控需求。 |

這些規則會在 SMF 中透過對 PCF 的 SBI 請求獲取「策略與 QoS 配置」，再結合 UE 的訂閱資料與目前網路狀態，轉譯成具體的 PDR/FAR/QER/URR 組合，最後以 PFCP 訊息下發到 UPF。換句話說，PCF 給的是「抽象的策略」，而 SMF 則負責把這些策略落實成 UPF 可以直接執行的資料平面規則。

## 本章小結

本章從高層介紹了 SMF 與 UPF 在 5GC 中的分工，以及它們透過 N3/N4、GTP-U、PFCP 與 gtp5g 等機制，將「PDU 會話與策略」具體落實成資料平面的轉送與計費行為。

你可以簡單記成：**SMF 想規則、UPF 做執行**，SMF 根據訂閱與策略組合出 PDR/FAR/QER/URR，UPF 則依此在 N3 上實際收送與轉發封包。

理解這一對「一文一武」的搭檔之後，接下來在閱讀其他 NF（例如 PCF、UDM、AUSF 等）時，就能更清楚地看出它們如何從各自角度影響整體的控制面與使用者面行為。

<div class="chapter-nav">
  <a href="../chapter6/" class="nav-btn nav-next" title="下一章：其他網路元件">
    <span class="arrow"></span>
  </a>
</div>
