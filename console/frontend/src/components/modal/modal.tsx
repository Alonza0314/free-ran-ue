import type { ReactNode } from 'react'
import styles from './modal.module.css'
import Button from '../button/button'

interface ModalProps {
  isOpen: boolean
  onClose: () => void
  title: string
  children: ReactNode
  onSubmit?: () => void
}

export default function Modal({ 
  isOpen, 
  onClose, 
  title, 
  children,
  onSubmit 
}: ModalProps) {
  if (!isOpen) return null

  return (
    <div className={styles.overlay}>
      <div className={styles.modal}>
        <div className={styles.header}>
          <h2 className={styles.title}>{title}</h2>
        </div>
        <div className={styles.body}>
          {children}
        </div>
        <div className={styles.footer}>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          {onSubmit && (
            <Button onClick={onSubmit}>
              Submit
            </Button>
          )}
        </div>
      </div>
    </div>
  )
}
