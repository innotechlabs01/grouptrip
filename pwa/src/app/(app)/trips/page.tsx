import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api';
import { useState } from 'react';
export default function TripsPage(){
  const qc = useQueryClient();
  const { data:trips=[] } = useQuery({ queryKey:['trips'], queryFn:()=>apiFetch('/trips') });
  const [title,setTitle]=useState('');
  const mut = useMutation({
    mutationFn:(t:string)=>apiFetch('/trips',{method:'POST',body:JSON.stringify({title:t})}),
    onSuccess:()=>qc.invalidateQueries({queryKey:['trips']})
  });
  return <div className="p-6"><h1 className="text-2xl font-bold">Trips</h1>
  <form onSubmit={e=>{e.preventDefault(); mut.mutate(title); setTitle('');}} style={{marginBottom:12}}>
    <input value={title} onChange={e=>setTitle(e.target.value)} placeholder="Nuevo viaje"/>
    <button type="submit">Crear</button>
  </form>
  <ul>{trips.map((t:any)=><li key={t.id}>{t.title} <button onClick={async()=>{ await apiFetch(`/trips/${t.id}/contribute`,{method:'POST'}); qc.invalidateQueries({queryKey:['trips']}); }}>Pagar</button></li>)}</ul>
  </div>
}