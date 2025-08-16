import { createContext, useContext, useState } from 'react'
import type { ReactNode } from 'react'

import type { ApiConsoleGnbInfoPost200Response } from '../api'

interface GnbConnection {
  ip: string
  port: number
}

interface GnbWithConnection extends ApiConsoleGnbInfoPost200Response {
  connection?: GnbConnection
}

interface GnbContextType {
  gnbList: GnbWithConnection[]
  addGnb: (gnb: ApiConsoleGnbInfoPost200Response, connection: GnbConnection) => { exists: boolean }
  removeGnb: (gnbId: string) => void
}

const GnbContext = createContext<GnbContextType | undefined>(undefined)

const GNB_STORAGE_KEY = 'gnbList'

function loadGnbList(): GnbWithConnection[] {
  const stored = localStorage.getItem(GNB_STORAGE_KEY)
  return stored ? JSON.parse(stored) : []
}

export function GnbProvider({ children }: { children: ReactNode }) {
  const [gnbList, setGnbList] = useState<GnbWithConnection[]>(loadGnbList())

  const addGnb = (gnb: ApiConsoleGnbInfoPost200Response, connection: GnbConnection) => {
    const existingIndex = gnbList.findIndex(item => 
      item.gnbInfo?.gnbId === gnb.gnbInfo?.gnbId ||
      (item.connection?.ip === connection.ip && item.connection?.port === connection.port)
    )

    let newList
    if (existingIndex !== -1) {
      newList = [...gnbList]
      newList[existingIndex] = { ...gnb, connection }
    } else {
      newList = [...gnbList, { ...gnb, connection }]
    }

    setGnbList(newList)
    localStorage.setItem(GNB_STORAGE_KEY, JSON.stringify(newList))
    
    return { exists: existingIndex !== -1 }
  }

  const removeGnb = (gnbId: string) => {
    const newList = gnbList.filter(gnb => gnb.gnbInfo?.gnbId !== gnbId)
    setGnbList(newList)
    localStorage.setItem(GNB_STORAGE_KEY, JSON.stringify(newList))
  }

  return (
    <GnbContext.Provider value={{ gnbList, addGnb, removeGnb }}>
      {children}
    </GnbContext.Provider>
  )
}

export function useGnb() {
  const context = useContext(GnbContext)
  if (context === undefined) {
    throw new Error('useGnb must be used within a GnbProvider')
  }
  return context
}
