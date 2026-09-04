import { useEffect, useState } from 'react';
import { apiFetch } from '@/lib/api';
export default function DecisionsPage(){
  const [dec,setDec]=useState<any[]>([]);
  const [q,setQ]=useState('');
  const load=()=>apiFetch('/decisions').then(setDec).catch(()=>{});
  useEffect(()=>{load()},[]);
  const create=async(e:React.FormEvent)=>{
    e.preventDefault();
    await apiFetch('/decisions',{method:'POST',body:JSON.stringify({question:q})});
    setQ(''); load();
  };
  return <div className="p-6"><h1 className="text-2xl font-bold">Decisiones</h1>
  <form onSubmit={create} style={{marginBottom:12}}><input value={q} onChange={e=>setQ(e.target.value)} placeholder="Nueva decisión"/><button type="submit">Crear</button></form>
  <ul>{dec.map(d=><li key={d.id}>{d.question}</li>)}</ul></div>
}