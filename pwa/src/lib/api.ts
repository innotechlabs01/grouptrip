export const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
export async function apiFetch(path:string, init?: RequestInit){
  const res = await fetch(`${API_BASE}${path}`, {...init, credentials:'include', headers:{'Content-Type':'application/json', ...(init?.headers||{})}});
  if(!res.ok) throw new Error(await res.text());
  return res.json();
}