import styles from './switch.module.css'

interface SwitchProps {
  checked: boolean
  onChange: (checked: boolean) => void
  label?: string
  disabled?: boolean
}

export default function Switch({ checked, onChange, label, disabled = false }: SwitchProps) {
  return (
    <label className={styles.switch}>
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        disabled={disabled}
      />
      <span className={styles.slider}></span>
      {label && <span className={styles.label}>{label}</span>}
    </label>
  )
}
