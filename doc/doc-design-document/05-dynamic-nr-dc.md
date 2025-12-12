# Dynamic NR-DC

> [!Note]
> Dynamic NR-DC means the DC feature (two tunnels in one PDU Session) will be trigger **after** the initial PDU session establishment procedure.

## At gNB

If gNB get the signal to modify the UE's tunnel, it will iinteract with secondary gNB for retrieving the second tunnel's info, like `TEID`.

- For master gNB:

    1. Build the target `PDUSessionResourceModifyIndicationTransfer` message.

        ```go
        pduSessionModifyIndicationTransfer, err := getPDUSessionResourceModifyIndicationTransfer(ranUe.GetDlTeid(), g.ranN3Ip, 1)
        ```

    2. Encapsulate into `PDUSessionResourceModifyIndication` NGAP message.

        ```go
        pduSessionModifyIndication, err := getPDUSessionResourceModifyIndication(ranUe.GetAmfUeId(), ranUe.GetRanUeId(), constant.PDU_SESSION_ID, pduSessionModifyIndicationTransfer)
        ```

    3. Interact with secondary gNB.

        ```go
        pduSessionModifyIndication, err = g.xnPduSessionResourceModifyIndication(ranUe.GetMobileIdentityIMSI(), pduSessionModifyIndication)
        ```

        This step will start communication with secondary gNB via Xn-interface.

    4. Send the modify indication message to AMF for core network DC setup.

        ```go
        n, err := g.n2Conn.Write(pduSessionModifyIndication)
        ```

    5. Receive the confirm message and transmit it to secondary gNB

        ```go
        _, err = g.xnPduSessionResourceModifyConfirm(ranUe.GetMobileIdentityIMSI(), ngapPduSessionResourceModifyConfirmRaw[:n])
        ```

    6. Send tunnel update message to UE

        ```go
        modifyMessage := []byte(util.TUNNEL_UPDATE)
        n, err = ranUe.GetN1Conn().Write(modifyMessage)
        ```

        This message is just designed by free-ran-ue, not followed 3GPP. The purpose of the message is letting UE update the data plane configuration.

- For secondary gNB:

    1. Receive the modify indication NGAP message and insert its tunnel's information.
    2. Receive the confirm NGAP message and update the uplink TEID.

    For more details implemtation, please refer to: [xn.go](https://github.com/free-ran-ue/free-ran-ue/blob/main/gnb/xn.go)

## At UE

After master gNB finishing the modify procedure, it will send an update message to UE. Once UE received this tunnel update message, it will update the data plane configuration.

- Modify from non-DC to DC:

    1. Dial a connection to secondary gNB's data plane connection.

        ```go
        conn, err := util.TcpDialWithOptionalLocalAddress(u.nrdc.dcRanDataPlane.ip, u.nrdc.dcRanDataPlane.port, u.nrdc.dcLocalDataPlaneIp)
        ```

    2. Start the data plane read for secondaru gNB.

        ```go
        buffer := make([]byte, 4096)
        for {
            n, err := u.dcRanDataPlaneConn.Read(buffer)
            u.readFromRan <- buffer[:n]
        }
        ```

- Modify from DC to non-DC

    1. Close the secondary gNB's data plane connection.

        ```go
        u.dcRanDataPlaneConn.Close()
        ```

The data plane handler will based on the `ue.nrdc.enable` flag to do the traffic flow split.
