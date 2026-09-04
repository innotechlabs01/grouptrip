import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api';
import { useState } from 'react';
export default function BudgetPage(){
  const qc = useQueryClient();
  const { data:cats=[] } = useQuery({ queryKey:['budget','categories'], queryFn:()=>apiFetch('/budget/categories') });
  const [name,setName]=useState('');
  const mut = useMutation({
    mutationFn:(n:string)=>apiFetch('/budget/categories',{method:'POST',body:JSON.stringify({name:n})}),
    onSuccess:()=>qc.invalidateQueries({queryKey:['budget','categories']})
  });
  const create=e=>{ e.preventDefault(); mut.mutate(name); setName(''); };
  return <div className="p-6"><h1 className="text-2xl font-bold">Budget</h1>
  <form onSubmit={create} style={{marginBottom:12}}><input value={name} onChange={e=>setName(e.target.value)} placeholder="Nueva categoría"/><button type="submit">Crear</button></form>
  <ul>{cats.map((c:any)=><li key={c.id}>{c.name}</li>)}</ul></div>
}