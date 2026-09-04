import { useEffect, useState } from 'react';
import { apiFetch } from '@/lib/api';
export default function BudgetPage(){
  const [cats,setCats]=useState<any[]>([]);
  const [name,setName]=useState('');
  const load=()=>apiFetch('/budget/categories').then(setCats).catch(()=>{});
  useEffect(()=>{load()},[]);
  const create=async(e:React.FormEvent)=>{
    e.preventDefault();
    await apiFetch('/budget/categories',{method:'POST',body:JSON.stringify({name})});
    setName(''); load();
  };
  return <div className="p-6"><h1 className="text-2xl font-bold">Budget</h1>
  <form onSubmit={create} style={{marginBottom:12}}><input value={name} onChange={e=>setName(e.target.value)} placeholder="Nueva categoría"/><button type="submit">Crear</button></form>
  <ul>{cats.map(c=><li key={c.id}>{c.name}</li>)}</ul></div>
}