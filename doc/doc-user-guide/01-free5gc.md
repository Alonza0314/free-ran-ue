# free5GC

> [!Note]
> For more free5GC's information, please refer to [free5GC's official website](https://free5gc.org/)

## A. Build and Install free5GC

Please refer to the official document: [Install free5GC](https://free5gc.org/guide/3-install-free5gc/)

## B. Start free5GC

1. Setting Parameters

    - There will be three YAML file need modified:

        - ~/free5gc/config/amfcfg.yaml
        - ~/free5gc/config/smfcfg.yaml
        - ~/free5gc/config/upfcfg.yaml

    - ~/free5gc/config/amfcfg.yaml

        Replace `ngapIpList` IP from `127.0.0.18` to you export IP:

        ```yaml
        ngapIpList:
          - <your export IP>
        ```

    - ~/free5gc/config/smfcfg.yaml

        Replace N3 interface's endpoints IP from `127.0.0.8` to your export IP:

        ```yaml
        interfaces:
          - interfaceType: N3
            endpoints:
              - <your export IP>
        ```

    - ~/free5gc/config/upfcfg.yaml

        Replace N6 interface address IP from `127.0.0.8` to your export IP:

        ```yaml
        gtpu:
          forwarder: gtp5g
          ifList:
            - addr: <your export IP>
        ```

2. Check IP Forward is enabled

    If you have rebooted your machine, remember to run these command with setting your export network interface:

    ```bash
    sudo sysctl -w net.ipv4.ip_forward=1
    sudo iptables -t nat -A POSTROUTING -o <export network interface> -j MASQUERADE
    sudo systemctl stop ufw
    sudo iptables -I FORWARD 1 -j ACCEPT
    ```

3. Execute `run.sh` in free5GC

    ```bash
    ./run.sh
    ```

## C. Create A Subscriber in free5GC

Please refer to the official document: [Create Subscriber via Webconsole](https://free5gc.org/guide/Webconsole/Create-Subscriber-via-webconsole/)
