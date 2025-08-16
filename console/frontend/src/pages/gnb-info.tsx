import { useParams, useNavigate } from 'react-router-dom'
import styles from './css/gnb-info.module.css'
import { useGnb } from '../context/gnbContext'
import Button from '../components/button/button'
import Sidebar from '../components/sidebar/sidebar'
import Switch from '../components/switch/switch'

export default function GnbInfo() {
  const { gnbId } = useParams()
  const navigate = useNavigate()
  const { gnbList } = useGnb()

  const gnbInfo = gnbList.find(gnb => gnb.gnbInfo?.gnbId === gnbId)?.gnbInfo

  if (!gnbInfo) {
    return (
      <div className={styles.container}>
        <Sidebar />
        <div className={styles.content}>
          <div className={styles.header}>
            <Button variant="secondary" onClick={() => navigate('/gnb')}>
              Back to List
            </Button>
          </div>
          <div className={styles.error}>
            gNB not found
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className={styles.container}>
      <Sidebar />
      <div className={styles.content}>
        <div className={styles.header}>
          <Button variant="secondary" onClick={() => navigate('/gnb')}>
            Back to List
          </Button>
        </div>

        <div className={styles.infoCard}>
          <h2 className={styles.title}>gNB Information</h2>
          
          <div className={styles.infoGroup}>
            <label>gNB ID</label>
            <div>{gnbInfo.gnbId}</div>
          </div>

          <div className={styles.infoGroup}>
            <label>gNB Name</label>
            <div>{gnbInfo.gnbName}</div>
          </div>

          <div className={styles.infoGroup}>
            <label>PLMN ID</label>
            <div>{gnbInfo.plmnId}</div>
          </div>

          <div className={styles.infoGroup}>
            <label>SNSSAI</label>
            <div>
              <div>SST: {gnbInfo.snssai?.sst || 'N/A'}</div>
              <div>SD: {gnbInfo.snssai?.sd || 'N/A'}</div>
            </div>
          </div>
        </div>

        <div className={styles.infoCard}>
          <h2 className={styles.title}>RAN UE List</h2>
          <div className={styles.ueList}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>No.</th>
                  <th>UE</th>
                  <th>DC-status</th>
                </tr>
              </thead>
              <tbody>
                {gnbInfo.ranUeList?.map((ue, index) => (
                  <tr key={ue.imsi}>
                    <td>{index + 1}</td>
                    <td>{ue.imsi}</td>
                    <td>
                      <Switch
                        checked={ue.nrdcIndicator || false}
                        onChange={() => {
                        }}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <div className={styles.infoCard}>
          <h2 className={styles.title}>XN UE List</h2>
          <div className={styles.ueList}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>No.</th>
                  <th>UE</th>
                </tr>
              </thead>
              <tbody>
                {gnbInfo.xnUeList?.map((ue, index) => (
                  <tr key={ue.imsi}>
                    <td>{index + 1}</td>
                    <td>{ue.imsi}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  )
}
