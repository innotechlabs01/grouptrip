import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api';
import { useState } from 'react';
export default function DecisionsPage(){
  const qc = useQueryClient();
  const { data:dec=[], isLoading, isError, error } = useQuery({ queryKey:['decisions'], queryFn:()=>apiFetch('/decisions') });
  const [title,setTitle]=useState('');
  const [options,setOptions]=useState('');
  const mut = useMutation({ mutationFn:(t:string,opts:string[])=>apiFetch('/decisions',{method:'POST',body:JSON.stringify({title:t,options:opts})}), onSuccess:()=>qc.invalidateQueries({queryKey:['decisions']}) });
  const voteMut = useMutation({ mutationFn:(id:string,opt:string)=>apiFetch(`/decisions/${id}/vote`,{method:'POST',body:JSON.stringify({option:opt})}), onSuccess:()=>qc.invalidateQueries({queryKey:['decisions']}) });
  const create=e=>{ e.preventDefault(); if(!title) return; mut.mutate(title, options.split(',').map(o=>o.trim())); setTitle(''); setOptions(''); };
  if(isLoading) return <div className="p-6">Cargando...</div>
  if(isError) return <div className="p-6 text-red-600">Error: {String(error)}</div>
  return <div className="p-6"><h1>Decisiones</h1>
  <form onSubmit={create} style={{marginBottom:12}}><input value={title} onChange={e=>setTitle(e.target.value)} placeholder="Título"/><input value={options} onChange={e=>setOptions(e.target.value)} placeholder="Opciones separadas por ,"/><button type="submit" disabled={mut.isPending}>Crear</button></form>
  <ul>{dec.map((d:any)=><li key={d.id}><strong>{d.title}</strong><ul>{(d.options||[]).map((o:string,i:number)=><li key={i}>{o} <button onClick={()=>voteMut.mutate(d.id,o)}>Votar</button></li>)}</ul></li>)}</ul></div>
}
