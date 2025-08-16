import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import styles from './css/login.module.css'
import logoImg from '../assets/free-ran-ue.jpg'
import { consoleApi } from '../apiCfg'
import ErrorBox from '../components/errorBox/errorBox'
import { useErrors } from '../hooks/useErrors'

export default function Login() {
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const { errors, addError, removeError } = useErrors()

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    try {
      const response = await consoleApi.apiConsoleLoginPost({
        username,
        password
      })

      if (response.data.token) {
        localStorage.setItem('token', response.data.token)
        navigate('/dashboard')
      }
    } catch (err: any) {
      addError(err.response?.data?.message || 'Login failed. Please try again.')
    }
  }

  return (
    <>
      <ErrorBox 
        errors={errors}
        onClose={removeError}
        duration={5000}
      />
      <div className={styles.loginContainer}>
        <div className={styles.loginHeader}>
          <img 
            src={logoImg}
            alt="free-ran-ue" 
            className={styles.logo}
          />
          <h1>Welcome to free-ran-ue</h1>
          <p>Please sign in to continue</p>
        </div>
        <form className={styles.loginCard} onSubmit={handleSubmit}>
          <div className={styles.formGroup}>
            <input
              className={styles.input}
              id="username"
              name="username"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="Username"
              required
              autoComplete="username"
            />
          </div>
          <div className={styles.formGroup}>
            <input
              className={styles.input}
              id="password"
              name="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Password"
              required
              autoComplete="current-password"
            />
          </div>
          <button className={styles.submitButton} type="submit">
            Sign In
          </button>
        </form>
      </div>
    </>
  )
}

