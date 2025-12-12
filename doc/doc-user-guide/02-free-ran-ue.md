# free-ran-ue

> [!Caution]
> Do not start the UE on the same machine as free5GC, as this will cause data plane forwarding failures.
>
> There are two options for deployment:
>
> - Use two separate machines.
> - Use namespace separation. For details, please refer to [Quick Start](03-quickstart-free-ran-ue.md).

## A. Prerequisites

- Golang:

    - free-ran-ue is built, tested and run with `go1.24.5 linux/amd64`
    - If Golang is not installed on your system, please execute the following commands:

        - Install Golang:

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

        - Check Installation. You should see the version information:

            ```bash
            go version
            ```

    - If another version of Golang is installed, please execute the following commands to replace it:

        ```bash
        sudo rm -rf /usr/local/go
        wget https://dl.google.com/go/go1.24.5.linux-amd64.tar.gz
        sudo tar -C /usr/local -zxvf go1.24.5.linux-amd64.tar.gz
        source $HOME/.bashrc
        go version
        ```

## B. Clone and Build free-ran-ue

- Clone

    ```bash
    git clone https://github.com/free-ran-ue/free-ran-ue.git
    ```

- Build

    ```bash
    cd free-ran-ue
    make
    ```

    After building, a binary executable file will be generated in the `build` folder.

## C. Start gNB

- Modify the configuration file for gNB:

    The configuration `YAML` file template is located at `free-ran-ue/config/gnb.yaml`.

    Ensure that the information matches your core network settings. For core network settings, please refer to: [Start free5GC](01-free5gc.md)

- Start gNB:

    After configuring the `YAML` file, execute the binary in the `build` folder to start gNB with the specified configuration file:

    ```bash
    ./build/free-ran-ue gnb -c config/gnb.yaml
    ```

## D. Start UE

- Modify the configuration file for UE:

    The configuration `YAML` file template is located at `free-ran-ue/config/ue.yaml`.

    Ensure that the information matches your web console settings, especially the `authenticationSubscription` section. For web console settings, please refer to: [Create Subscriber via Webconsole](https://free5gc.org/guide/Webconsole/Create-Subscriber-via-webconsole/)

    Pay attention to the `ueTunnelDevice` field, as this will be the name of the network interface created later.

- Start UE:

    After configuring the `YAML` file, execute the binary in the `build` folder to start UE with the specified configuration file:

    ```bash
    ./build/free-ran-ue ue -c config/ue.yaml
    ```

## E. ICMP Test

After UE has started, a network interface will be available. Use `ifconfig` to check it:

```bash
ifconfig
```

Expected output included:

```bash
ueTun0: flags=4305<UP,POINTOPOINT,RUNNING,NOARP,MULTICAST>  mtu 1500
        inet 10.60.0.1  netmask 255.255.255.255  destination 10.60.0.1
        inet6 fe80::b1e9:2933:3c64:b981  prefixlen 64  scopeid 0x20<link>
        unspec 00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00  txqueuelen 500  (UNSPEC)
        RX packets 0  bytes 0 (0.0 B)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 3  bytes 144 (144.0 B)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0
```

ICMP test with `ueTun0`:

```bash
ping -I ueTun0 8.8.8.8 -c 5
```

Expected successful output:

```bash
PING 8.8.8.8 (8.8.8.8) from 10.60.0.1 ueTun0: 56(84) bytes of data.
64 bytes from 8.8.8.8: icmp_seq=1 ttl=116 time=3.99 ms
64 bytes from 8.8.8.8: icmp_seq=2 ttl=116 time=3.90 ms
64 bytes from 8.8.8.8: icmp_seq=3 ttl=116 time=3.84 ms
64 bytes from 8.8.8.8: icmp_seq=4 ttl=116 time=4.07 ms
64 bytes from 8.8.8.8: icmp_seq=5 ttl=116 time=3.62 ms

--- 8.8.8.8 ping statistics ---
5 packets transmitted, 5 received, 0% packet loss, time 4007ms
rtt min/avg/max/mdev = 3.618/3.881/4.067/0.152 ms
```

Now, both gNB and UE are running seccussfully.
