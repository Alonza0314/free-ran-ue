# free-ran-ue

![free-ran-ue](image/free-ran-ue.jpg)

## **Introduction**

**free-ran-ue** is an open source project developed by [Alonza0314](https://github.com/Alonza0314).

The primary goal of this project is to create a next-generation RAN/UE simulator for testing the [NR-DC (New Radio Dual Connectivity)](https://free5gc.org/blog/20250219/20250219/) feature in [free5GC](https://free5gc.org/).

A key technical innovation of free-ran-ue is the implementation of [userspace GTP-U](doc-design-document/01-userspace-gtp-u.md), replacing the traditional kernel-space GTP5G. This approach provides greater flexibility and control over GTP packet manipulation, making it easier to manage and customize data plane traffic.

## **Support Features**

- Fundamental support for **UE registration** and **PDU session establishment** in both control and data planes
- **Advanced feature**: Support for [NR-DC (New Radio Dual Connectivity)](https://free5gc.org/blog/20250219/20250219/), enabling simultaneous data plane connections across two gNBs
- **Multiple UEs**: Launch multiple UEs with a single command.

For more information on how to use **free-ran-ue**, please refer to the [User Guide](doc-user-guide/index.md).
