export const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';
export async function apiFetch(path:string, init?: RequestInit){
  const res = await fetch(`${API_BASE}${path}`, init);
  if(!res.ok) throw new Error(`API ${res.status}`);
  return res.json();
}
