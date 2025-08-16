import { useEffect } from 'react'
import styles from './errorBox.module.css'

interface ErrorMessage {
  id: string
  message: string
  timestamp: number
}

interface ErrorBoxProps {
  errors: ErrorMessage[]
  onClose: (id: string) => void
  duration?: number
}

function SingleError({ 
  error, 
  onClose, 
  duration = 5000,
  index
}: { 
  error: ErrorMessage
  onClose: (id: string) => void
  duration: number
  index: number
}) {
  useEffect(() => {
    const timer = setTimeout(() => {
      onClose(error.id)
    }, duration)
    return () => clearTimeout(timer)
  }, [error.id, duration, onClose])

  return (
    <div 
      className={styles.errorBox} 
      style={{ 
        top: `${(index * 4.5) + 1}rem`,
        animationDelay: `${index * 0.05}s`
      }}
    >
      <span className={styles.message}>{error.message}</span>
      <button 
        type="button" 
        className={styles.closeButton}
        onClick={() => onClose(error.id)}
        aria-label="Close error message"
      >
        x
      </button>
    </div>
  )
}

export default function ErrorBox({ errors, onClose, duration = 5000 }: ErrorBoxProps) {
  return (
    <div className={styles.errorBoxContainer}>
      {errors.map((error, index) => (
        <SingleError
          key={error.id}
          error={error}
          onClose={onClose}
          duration={duration}
          index={index}
        />
      ))}
    </div>
  )
}
