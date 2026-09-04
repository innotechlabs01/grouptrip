import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api';
import { useState } from 'react';
export default function DecisionsPage(){
  const qc = useQueryClient();
  const { data:dec=[] } = useQuery({ queryKey:['decisions'], queryFn:()=>apiFetch('/decisions') });
  const [q,setQ]=useState('');
  const mut = useMutation({
    mutationFn:(question:string)=>apiFetch('/decisions',{method:'POST',body:JSON.stringify({question})}),
    onSuccess:()=>qc.invalidateQueries({queryKey:['decisions']})
  });
  const create=e=>{ e.preventDefault(); mut.mutate(q); setQ(''); };
  return <div className="p-6"><h1 className="text-2xl font-bold">Decisiones</h1>
  <form onSubmit={create} style={{marginBottom:12}}><input value={q} onChange={e=>setQ(e.target.value)} placeholder="Nueva decisión"/><button type="submit">Crear</button></form>
  <ul>{dec.map((d:any)=><li key={d.id}>{d.question}</li>)}</ul></div>
}