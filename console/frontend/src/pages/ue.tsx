import { Outlet } from 'react-router-dom'
import styles from './css/ue.module.css'
import Sidebar from '../components/sidebar/sidebar'

export default function Ue() {
  return (
    <div className={styles.container}>
      <Sidebar />
      <div className={styles.content}>
        <div className={styles.header}>
          <h1>UE</h1>
        </div>
        <Outlet />
      </div>
    </div>
  )
}
