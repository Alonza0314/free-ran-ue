import { Outlet, useNavigate } from 'react-router-dom'
import styles from './css/dashboard.module.css'
import Sidebar from '../components/sidebar/sidebar'
import StatsCard from '../components/stats/stats-card'
import { useGnb } from '../context/gnbContext'

export default function Dashboard() {
  const navigate = useNavigate()
  const { gnbList } = useGnb()

  return (
    <div className={styles.container}>
      <Sidebar />
      <div className={styles.content}>
        <div className={styles.header}>
          <h1>Dashboard</h1>
        </div>

        <div className={styles.stats}>
          <div className={styles.statsCard} onClick={() => navigate('/gnb')}>
            <StatsCard 
              title="Total gNBs"
              value={gnbList.length}
              description="Click to view all gNBs"
            />
          </div>
        </div>

        <Outlet />
      </div>
    </div>
  )
}
