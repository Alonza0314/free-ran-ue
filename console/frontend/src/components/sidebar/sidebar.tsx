import { NavLink } from 'react-router-dom'
import styles from './sidebar.module.css'

export default function Sidebar() {
  const navItems = [
    { path: '/dashboard', label: 'Dashboard' },
    { path: '/gnb', label: 'gNB' },
    { path: '/ue', label: 'UE' },
  ]

  return (
    <aside className={styles.sidebar}>
      <div className={styles.logo}>
        <h1>free-ran-ue</h1>
      </div>
      <nav className={styles.nav}>
        {navItems.map(item => (
          <NavLink
            key={item.path}
            to={item.path}
            className={({ isActive }) => 
              `${styles.navItem} ${isActive ? styles.active : ''}`
            }
          >
            {item.label}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
