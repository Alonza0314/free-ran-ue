import styles from './button.module.css'

interface ButtonProps {
  children: React.ReactNode
  onClick?: () => void
  variant?: 'primary' | 'secondary'
  type?: 'button' | 'submit' | 'reset'
}

export default function Button({ 
  children, 
  onClick, 
  variant = 'primary',
  type = 'button'
}: ButtonProps) {
  return (
    <button
      type={type}
      className={`${styles.button} ${styles[variant]}`}
      onClick={onClick}
    >
      {children}
    </button>
  )
}
