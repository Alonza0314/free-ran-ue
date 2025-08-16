import { useState, useCallback } from 'react'

interface ErrorItem {
  id: string
  message: string
  timestamp: number
}

export function useErrors() {
  const [errors, setErrors] = useState<ErrorItem[]>([])

  const addError = useCallback((message: string) => {
    const id = Date.now().toString()
    setErrors(prev => [...prev, {
      id,
      message,
      timestamp: Date.now()
    }])
    return id
  }, [])

  const removeError = useCallback((id: string) => {
    setErrors(prev => {
      const newErrors = prev.filter(error => error.id !== id)
      return newErrors
    })
  }, [])

  return {
    errors,
    addError,
    removeError
  }
}
