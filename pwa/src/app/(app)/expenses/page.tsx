import { useEffect, useState } from 'react';
import { apiFetch } from '@/lib/api';
export default function ExpensesPage(){
  const [exp,setExp]=useState<any[]>([]);
  const [desc,setDesc]=useState('');
  const [amt,setAmt]=useState('');
  const load=()=>apiFetch('/expenses').then(setExp).catch(()=>{});
  useEffect(()=>{load()},[]);
  const create=async(e:React.FormEvent)=>{
    e.preventDefault();
    await apiFetch('/expenses',{method:'POST',body:JSON.stringify({description:desc,amount:Number(amt)})});
    setDesc(''); setAmt(''); load();
  };
  return <div className="p-6"><h1 className="text-2xl font-bold">Gastos</h1>
  <form onSubmit={create} style={{marginBottom:12,display:'flex',gap:8}}><input value={desc} onChange={e=>setDesc(e.target.value)} placeholder="Descripción"/><input value={amt} onChange={e=>setAmt(e.target.value)} placeholder="Monto"/><button type="submit">Agregar</button></form>
  <ul>{exp.map(e=><li key={e.id}>{e.description} - {e.amount}</li>)}</ul></div>
}