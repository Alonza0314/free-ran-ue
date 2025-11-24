# 5G 核心網輕鬆學：free5GC 與 RAN/UE 模擬器

> [!Important]
> 歡迎來到《5G Core Book》！
>
> 本書會用最有效率的方式帶領讀者搞懂 **5G 核心網路**、**部署 free5GC** 以及打造自己的 **RAN/UE 模擬器**！讓即使是新手小白也可以成為走在科技前端的 5G 核心網路大師！

## 為什麼會有這本書？

筆者在剛踏入 5G 領域時，總覺得整個世界非常抽象。標準文件裡充滿了大量介面、程序與縮寫：AMF、SMF、UPF、N1/N2/N3、PDU Session……

對於許多初學者而言，在閱讀 3GPP 規格或試圖理解投影片時，往往會遇到幾個共同的痛點：

- **名詞縮寫太多**：看到一長串縮寫，卻很難立即對應到它們的功能。
- **概念太抽象**：知道網元名稱，卻無法連結到實際封包與真實流程。
- **3GPP 協定過於龐大**：規格內容極其詳細且複雜，讓人難以找到下手點。

因此，筆者希望透過最簡單、最直觀、最實作導向的方式，帶領讀者進入 5G 的世界，並一步步實現屬於自己的 5G 核心網路環境。

---

## 這本書的內容適合誰？

- **剛入門 5G 的學生或工程師**，需要一個有脈絡的學習路線。
- **已經了解 4G/EPC**，想搞懂 5GC 新增了什麼、改了什麼。
- **對 free5GC 有興趣**，想看懂實作架構與各模組的角色。
- **想要一套可重現的 5G 環境**，包含核心網與 RAN/UE 模擬器。

不需要一開始就很熟 3GPP 規格，但如果你有基本的網路（IP/TCP/UDP）、Linux 操作、Docker 或 Kubernetes 經驗，會讀得更順！

---

## 本書的結構與閱讀建議

本系列大致分為四個部分（可依需求跳讀）：

- [**Part I — 5G 核心網路的基礎與演進背景**](./part1-history/chapter1)

    從 4G EPC 到 5G Core 的演變、3GPP 對 5GC 的整體視角，幫你建立大地圖。

- [**Part II — free5GC 全架構深度解析**](./part2-free5gc/chapter3)

    逐一拆解 AMF、SMF、UPF 等核心網元，對照實際模組與流程。

- [**Part III — free5GC 實戰部署與常見問題**](./part3-deploy/chapter8)

    從單機環境到多節點、多 UPF 的部署模式（docker / k8s），並整理常見坑與排錯思路。

- [**Part IV — free-ran-ue 模擬器**](./part4-free-ran-ue/chapter11)

    說明為什麼需要 RAN/UE 模擬器、free-ran-ue 的設計理念，以及如何與 free5GC 整合與進行實驗。

<!-- - [**Part V - 專題研究與延伸議題**] -->

**如果你是第一次接觸 5G：**

1. 建議先完整看完 **Part I**，建立整體架構概念。
2. 再依興趣選擇 **Part II / Part III** 做深入或實作。
3. 最後閱讀 **Part IV**，用 free-ran-ue 實際跑一輪 End-to-End 測試。

---

## 實驗環境與開源專案

本系列會大量使用以下開源專案：

- **free5GC**：開源 5G Core 專案

    官方網站與原始碼請參考 [free5GC 官方資源](https://www.free5gc.org/) 與其 GitHub。

- **free-ran-ue**：本專案提供的 RAN/UE 模擬器  

    原始碼與文件位於 [GitHub](https://github.com/Alonza0314/free-rna-ue)。

文中所有實驗步驟都盡量做到：

- **可重現**：同樣的指令與設定，你也能在自己的環境跑起來。
- **逐步解釋**：不只給指令，也說明為什麼要這樣做、背後對應到哪個網元或流程。

<div class="chapter-nav">
  <a href="author/" class="nav-btn nav-next" title="下一章：作者介紹">
    <span class="arrow"></span>
  </a>
</div>
