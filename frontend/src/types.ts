export type DriverName =
  | 'mysql'
  | 'postgres'
  | 'mssql'
  | 'mongodb'
  | 'redis'
  | 'etcd'
  | 'sqlite'
  | 'sqlite-bridge'

export interface DSN {
  id: string
  name: string
  driver: DriverName
  dsn: string
  created_at: string
  updated_at: string
}

export interface Bridge {
  id: string
  name: string
  type: string
  connected: boolean
  connected_at?: string
  secret?: string
  created_at: string
  updated_at: string
}

export interface Alert {
  id: number
  message: string
  type: 'success' | 'error'
}
