import { Outlet, useNavigate } from 'react-router-dom'
import { useState } from 'react'
import styles from './css/gnb.module.css'
import Sidebar from '../components/sidebar/sidebar'
import Button from '../components/button/button'
import AddGnbModal from '../components/gnb/add-gnb-modal'
import { useGnb } from '../context/gnbContext'

export default function Gnb() {
  const [isAddModalOpen, setIsAddModalOpen] = useState(false)
  const { gnbList, removeGnb } = useGnb()
  const navigate = useNavigate()

  const handleAddGnb = async () => {
    setIsAddModalOpen(true)
  }

  return (
    <div className={styles.container}>
      <Sidebar />
      <div className={styles.content}>
        <div className={styles.header}>
          <h1>gNB</h1>
          <Button onClick={handleAddGnb}>Add gNB</Button>
        </div>

        <div className={styles.list}>
          <table className={styles.table}>
            <thead className={styles.tableHeader}>
              <tr>
                <th>No.</th>
                <th>Status</th>
                <th>gNB Name</th>
                <th>Info</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody className={styles.tableBody}>
              {gnbList.map((gnb, index) => (
                <tr key={gnb.gnbInfo?.gnbId}>
                  <td>{index + 1}</td>
                  <td>
                    <span className={`${styles.status} ${styles.statusActive}`}>
                      Active
                    </span>
                  </td>
                  <td>{gnb.gnbInfo?.gnbName}</td>
                  <td>
                    <button 
                      className={`${styles.actionButton} ${styles.infoButton}`}
                      onClick={() => navigate(`/gnb/${gnb.gnbInfo?.gnbId}`)}
                    >
                      View Info
                    </button>
                  </td>
                  <td>
                    <button 
                      className={`${styles.actionButton} ${styles.removeButton}`}
                      onClick={() => removeGnb(gnb.gnbInfo?.gnbId || '')}
                    >
                      Remove
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <Outlet />
      </div>
      <AddGnbModal 
        isOpen={isAddModalOpen}
        onClose={() => setIsAddModalOpen(false)}
      />
    </div>
  )
}
