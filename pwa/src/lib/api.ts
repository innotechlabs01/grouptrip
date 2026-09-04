import { getToken, setToken, clearToken } from './auth'
export const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
async function refresh(){
  const res = await fetch(`${API_BASE}/auth/refresh`, { method:'POST', credentials:'include' })
  if(!res.ok) throw new Error('refresh failed')
  const data = await res.json()
  setToken(data.access_token)
  return data.access_token
}
export async function apiFetch(path:string, init?: RequestInit){
  const token = getToken()
  let res = await fetch(`${API_BASE}${path}`, { ...init, credentials:'include', headers:{ 'Content-Type':'application/json', ...(token?{'Authorization':`Bearer ${token}`}:{}), ...(init?.headers||{}) } })
  if(res.status===401){
    try{ const newTok = await refresh(); res = await fetch(`${API_BASE}${path}`, { ...init, credentials:'include', headers:{ 'Content-Type':'application/json', 'Authorization':`Bearer ${newTok}`, ...(init?.headers||{}) } }) }catch{ clearToken(); window.location.href='/login'; throw new Error('Unauthorized') }
  }
  if(!res.ok) throw new Error(await res.text())
  return res.json()
}
