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
  connected?: boolean
  created_at: string
  updated_at: string
}

export interface Alert {
  id: number
  message: string
  type: 'success' | 'error'
}
