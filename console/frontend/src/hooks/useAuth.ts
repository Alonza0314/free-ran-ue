import { consoleApi } from '../apiCfg'
import { useNavigate } from 'react-router-dom'

export function useAuthApi() {
  const navigate = useNavigate()

  const authenticate = async (addError?: (error: string) => void) => {
    const token = localStorage.getItem('token')
    
    if (!token) {
      if (addError) {
        addError('No authentication token found')
      }
      navigate('/login')
      return false
    }

    try {
      await consoleApi.apiConsoleAuthenticatePost({
        headers: {
          'Authorization': token
        }
      })
      return true
    } catch (error: any) {
      if (addError) {
        addError(error.response?.data?.message || 'Authentication failed')
      }
      localStorage.removeItem('token')
      navigate('/login')
      return false
    }
  }

  return { authenticate }
}
