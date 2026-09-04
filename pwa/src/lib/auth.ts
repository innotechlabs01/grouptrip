import { API_BASE } from './api';
export async function login(email:string,password:string){
  const res = await fetch(`${API_BASE}/auth/login`, {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({email,password}), credentials:'include'});
  if(!res.ok) throw new Error('Login failed');
  return res.json();
}
export async function register(email:string,password:string,name:string){
  const res = await fetch(`${API_BASE}/auth/register`, {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({email,password,name}), credentials:'include'});
  if(!res.ok) throw new Error('Register failed');
  return res.json();
}
export async function me(){
  const res = await fetch(`${API_BASE}/auth/me`, {credentials:'include'});
  if(res.status===401) return null;
  if(!res.ok) throw new Error('Me failed');
  return res.json();
}
