'use client';
import { useState } from 'react';
import { register } from '../../lib/auth';
export default function RegisterPage(){
  const [email,setEmail]=useState('');
  const [password,setPassword]=useState('');
  const [name,setName]=useState('');
  const [err,setErr]=useState('');
  async function onSubmit(e:React.FormEvent){
    e.preventDefault(); setErr('');
    try{ await register(email,password,name); location.href='/trips'; }catch(err:any){ setErr(err.message); }
  }
  return <main className="p-6 max-w-sm"><h1 className="text-2xl font-bold mb-4">Registro</h1><form onSubmit={onSubmit} className="space-y-3"><input className="border p-2 w-full" placeholder="nombre" value={name} onChange={e=>setName(e.target.value)} /><input className="border p-2 w-full" placeholder="email" value={email} onChange={e=>setEmail(e.target.value)} /><input type="password" className="border p-2 w-full" placeholder="password" value={password} onChange={e=>setPassword(e.target.value)} /><button className="bg-blue-600 text-white p-2 w-full">Crear cuenta</button></form>{err && <p className="text-red-600 mt-2">{err}</p>}</main>
}
