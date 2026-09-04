import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api';
import { useState } from 'react';
export default function ExpensesPage(){
  const qc = useQueryClient();
  const { data:exp=[], isLoading, isError, error } = useQuery({ queryKey:['expenses'], queryFn:()=>apiFetch('/expenses') });
  const { data:cats=[] } = useQuery({ queryKey:['budget','categories'], queryFn:()=>apiFetch('/budget/categories') });
  const [title,setTitle]=useState('');
  const [amount,setAmount]=useState('');
  const [catId,setCatId]=useState('');
  const mut = useMutation({
    mutationFn:()=>apiFetch('/expenses',{method:'POST',body:JSON.stringify({title,amount:parseFloat(amount),category_id:catId})}),
    onSuccess:()=>{qc.invalidateQueries({queryKey:['expenses']});setTitle('');setAmount('');setCatId('');}
  });
  const create=e=>{ e.preventDefault(); if(!title||!amount||!catId) return; mut.mutate(); };
  if(isLoading) return <div className="p-6">Cargando...</div>
  if(isError) return <div className="p-6 text-red-600">Error: {String(error)}</div>
  return <div className="p-6"><h1>Gastos</h1>
  <form onSubmit={create} style={{marginBottom:12}}>
    <select value={catId} onChange={e=>setCatId(e.target.value)}><option value="">Categoría</option>{cats.map((c:any)=><option key={c.id} value={c.id}>{c.name}</option>)}</select>
    <input value={title} onChange={e=>setTitle(e.target.value)} placeholder="Título"/>
    <input value={amount} onChange={e=>setAmount(e.target.value)} placeholder="Monto" type="number"/>
    <button type="submit" disabled={mut.isPending}>Crear</button>
  </form>
  {mut.isError && <div className="text-red-600">Error al crear</div>}
  <ul>{exp.map((e:any)=><li key={e.id}>{e.title} - ${e.amount} [{e.category_id}]</li>)}</ul></div>
}
