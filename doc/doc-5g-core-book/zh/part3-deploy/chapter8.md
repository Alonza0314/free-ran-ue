# 第 8 章：free5GC 部署實戰

free5GC 的[官方網站](https://free5gc.org) 提供了一系列的部署教學，這邊會針對實體機（包含 VM）部署以及 Docker 容器虛擬化部署方式做講解，希望透過解說如何設定 NF 的設定檔以及正確的編譯方式讓讀者不只學會單純的部署，進一步了解設定這些值的原因，使得未來再使用 free5GC 做相關研究時可以更輕鬆的調整部署方式。

> [!Note]
> 本書預設環境皆為 Ubuntu 24.04

## 8.1 GTP5G 核心模組安裝

### 8.1.1 GTP5G 安裝

gtp5g 的 Linux 核心模組是 free5GC 部署中非常關鍵的一個環節。無論是使用實體機（包含 VM）部署、以 Docker 方式啟動 UPF，甚至未來採用更進階的 K8s 架構，我們都必須先在**實際承載 UPF 流量的主機**上安裝 gtp5g（包含運行 UPF 容器的宿主機）。

原因在於：gtp5g 是一個安裝在 Linux kernel 內的模組，UPF 本身並不直接處理所有 GTP-U 封包，而是透過呼叫這個核心模組來完成封包的封裝、解封與轉送。

1. 安裝相關包

    ```bash
    sudo apt -y update
    sudo apt -y install git gcc g++ cmake autoconf libtool pkg-config libmnl-dev libyaml-dev
    ```

2. 安裝 GTP5G

    ```bash
    git clone -b v0.9.14 https://github.com/free5gc/gtp5g.git
    cd gtp5g
    make
    sudo make install
    ```

### 8.1.2 封包轉送

有鑑於我們會需要核心模組直接幫我們轉送資料平面的封包，因此我們需要在運行 UPF 的主機上開啟 IP Forward 的功能。

1. 關閉防火牆

    ```bash
    sudo systemctl stop ufw
    sudo systemctl disable ufw
    ```

2. 開啟轉送功能

    這邊需要注意第二個指令的`dn_interface`欄位需要替換成主機對外網卡的名稱

    ```bash
    sudo sysctl -w net.ipv4.ip_forward=1
    sudo iptables -t nat -A POSTROUTING -o <dn_interface> -j MASQUERADE
    sudo iptables -A FORWARD -p tcp -m tcp --tcp-flags SYN,RST SYN -j TCPMSS --set-mss 1400
    ```

## 8.2 實體機部署（包含 VM 部署）

### 8.2.1 安裝 Go 語言

> [!Note]
> 目前 free5GC 官方使用的 Go 版本是 1.24.5

1. 安裝 Go

    ```bash
    wget https://dl.google.com/go/go1.24.5.linux-amd64.tar.gz
    sudo tar -C /usr/local -zxvf go1.24.5.linux-amd64.tar.gz
    mkdir -p ~/go/{bin,pkg,src}
    # The following assume that your shell is bash:
    echo 'export GOPATH=$HOME/go' >> ~/.bashrc
    echo 'export GOROOT=/usr/local/go' >> ~/.bashrc
    echo 'export PATH=$PATH:$GOPATH/bin:$GOROOT/bin' >> ~/.bashrc
    echo 'export GO111MODULE=auto' >> ~/.bashrc
    source ~/.bashrc
    ```

2. 確認安裝

    ```bash
    go version
    ```

    如果可以看到版本號，就代表安裝成功了！

### 8.2.2 取得並編譯 free5GC 源碼

1. 從 Github 取得官方源碼

    ```bash
    git clone -j `nproc` --recursive https://github.com/free5gc/free5gc
    ```

    > [!Note]
    > -j `nproc` 的參數是希望使用平行加速的方式下載！
    >
    > --recursive 的參數是用來下載除了 free5GC 這個專案包裡面的程式嗎以外，其他一子模組方式存在的 NF 包

2. 編譯

    ```bash
    cd free5gc
    make
    ```

3. 修改設定檔

    在將 free5GC 跑起來之前，我們有三個 NF 的設定檔需要做修改，分別是：AMF、SMF、UPF，修改的目的是讓跑起來的核心網路可以被外面的 gNB 連上。

    - AMF（~/free5gc/config/amfcfg.yaml）

        將 `ngapIpList` 換成部署機器的對外網卡 IP，目的是要讓 gNB 可以透過這個 IP 對 AMF 做連線。

        ```yaml
        ngapIpList:
          - <your export IP>
        ```

    - SMF（~/free5gc/config/smfcfg.yaml）

        將 `interfaces` 中為 N3 的端點 IP 換成 UPF 開給 gNB 資料平面的網卡 IP，使得 SMF 可以告知 gNB 要對哪個 IP 做資料平面的連線。

        ```yaml
        interfaces:
          - interfaceType: N3
            endpoints:
              - <your export IP>
        ```

    - UPF（~/free5gc/config/upfcfg.yaml）

        將 `gtpu` 中的 `ifList` 換成 UPF 對外網卡的 IP，使 gNB 可以透過這個 IP 與 UPF 做連線。

        ```yaml
        gtpu:
          forwarder: gtp5g
          ifList:
            - addr: <your export IP>
        ```

4. 執行

    完成設定檔的修改後，就可以使用在 free5GC 專案下的執行腳本來一鍵啟動整個核網。

    ```bash
    cd free5gc
    ./run.sh
    ```

    在跑起來之後會看到很多的日誌記錄，如果沒有看到 ERROR 的記錄，就代表成功啟動了一個核心網路！

## 8.3 Docker 容器虛擬化部署

free5GC 除了實體執行以外，亦提供了 Docker 的 Compose 檔案讓使用者可以快速執行核網，同時又分為兩種情況：使用官方鏡像、自己本地建構鏡像。

> [!Caution]
> 使用前請透過 [Docker官網](https://docs.docker.com/engine/install/) 安裝 Docker Engine

### 8.3.1 使用官方鏡像

1. 取得 free5GC 的 Compose 專案

    ```bash
    git clone https://github.com/free5gc/free5gc-compose
    ```

2. 執行 Compose 啟動

    ```bash
    cd free5gc-compose
    docker compose -f docker-compose.yaml up
    ```

    等待鏡像拉取完成後就會自動啟動整個核網，核網中的每一 NF 都會是由一個容器運行。

3. 結束 Compose

    這邊可以直接按 Ctrl+c，然後會看到容器停止的畫面，但仍然需要使用 `down` 指令完全關閉 Compose。

    ```bash
    docker compose -f docker-compose.yaml down
    ```

### 8.3.2 本地自建鏡像

1. 取得 free5GC 的 Compose 專案

    ```bash
    git clone https://github.com/free5gc/free5gc-compose
    ```

2. 進入 `base` 目錄後取得想要建構鏡像的 free5GC 源碼

    ```bash
    cd free5gc-compose/base
    git clone -j `nproc` --recursice https://github.com/free5gc/free5gc
    ```

3. 使用 `make` 指令建構鏡像

    ```bash
    cd free5gc-compose
    make all
    ```

4. 使用本地鏡像執行 Compose

    ```bash
    cd free5gc-compose
    docker compose -f docker-compose-build.yaml up
    ```

5. 結束 Compose

    ```bash
    docker compose -f docker-compose-build.yaml down
    ```

## 8.4 本章小結

本章從實作角度帶著讀者完成一次完整的 free5GC 部署流程。首先，我們說明了為什麼必須在實際承載 UPF 流量的主機上安裝 gtp5g 核心模組，並示範如何啟用 IP Forward 與基本的 NAT/MTU 設定，讓 UPF 能夠正確轉送使用者面封包。接著，我們介紹了在實體機（或 VM）環境中安裝 Go、取得並編譯 free5GC 原始碼，以及調整 AMF/SMF/UPF 設定檔中與外部 gNB 連線相關的關鍵欄位，最後透過 `run.sh` 一鍵啟動整個核心網。

在此基礎上，本章也示範了如何使用 Docker Compose 以官方鏡像或本地自行建構的鏡像啟動 free5GC，讓讀者可以依照自身需求選擇「裸機部署」或「容器化部署」兩種方式。掌握了這些步驟與背後的原因之後，你應該已經能夠在自己的實驗環境中反覆重建與調整 free5GC，作為後續進一步研究 RAN、UE 或進階網路場景的基礎。

<div class="chapter-nav">
  <a href="../chapter9/" class="nav-btn nav-next" title="下一章：多節點 / 多 UPF 部署">
    <span class="arrow"></span>
  </a>
</div>
