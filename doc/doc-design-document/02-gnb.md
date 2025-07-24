# gNB (RAN)

> [!Note]
> The gNB (gNodeB) is a 5G base station that handles message transfer between the core network and User Equipment (UE). It serves as the radio access network node in 5G networks.

## Architecture Overview

![free-ran](../image/free-ran.png)

## Basic Connections

### Connections

The gNB (RAN) establishes four fundamental connections:

- **To Core Network:**

    - **Control Plane**: NGAP (Next Generation Application Protocol) with AMF (Access and Mobility Management Function)
    - **Data Plane**: GTP-U (GPRS Tunneling Protocol - User Plane) with UPF (User Plane Function)

- **To UE (User Equipment):**

    - **Control Plane**: NAS (Non-Access Stratum) signaling with UE
    - **Data Plane**: Raw packet transmission with UE over the radio interface

### Procedures

1. **NG Setup with AMF** (port: `38412`)

    - The gNB establishes a connection with the AMF to set up basic operational parameters and register itself with the core network.
    - This procedure includes exchanging supported features, served PLMNs (Public Land Mobile Networks), and TAC (Tracking Area Code) information.

2. **GTP Tunnel Establishment with UPF** (port: `2152`)

    - The gNB establishes GTP-U tunnels with the UPF for user data forwarding.
    - These tunnels are identified by TEID (Tunnel Endpoint Identifier) values for proper packet routing.

3. **UE Connection Management**

    Upon receiving a new UE control plane connection, the gNB initiates the following procedures:

    - **UE Registration**: Authenticates and registers the UE with the network
    - **PDU Session Establishment**: Creates data sessions for the UE's communication needs

## Xn Interface

In the current implementation, the Xn interface is specifically designed for exchanging TEID information to support the [NR-DC (New Radio Dual Connectivity)](https://free5gc.org/blog/20250219/20250219/) feature

## GTP Forwarding

The gNB implements GTP-U forwarding to handle user plane data between the UE and the core network. This includes:

- Encapsulation and decapsulation of user packets
- TEID-based routing

For more detailed information about GTP-U implementation, please refer to: [Userspace GTP-U](01-userspace-gtp-u.md)
