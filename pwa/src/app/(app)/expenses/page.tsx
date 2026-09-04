import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api';
import { useState } from 'react';
export default function ExpensesPage(){
  const qc = useQueryClient();
  const { data:exp=[] } = useQuery({ queryKey:['expenses'], queryFn:()=>apiFetch('/expenses') });
  const [desc,setDesc]=useState('');
  const [amt,setAmt]=useState('');
  const mut = useMutation({
    mutationFn:(payload:{description:string,amount:number})=>apiFetch('/expenses',{method:'POST',body:JSON.stringify(payload)}),
    onSuccess:()=>qc.invalidateQueries({queryKey:['expenses']})
  });
  const create=e=>{ e.preventDefault(); mut.mutate({description:desc,amount:Number(amt)}); setDesc(''); setAmt(''); };
  return <div className="p-6"><h1 className="text-2xl font-bold">Gastos</h1>
  <form onSubmit={create} style={{marginBottom:12,display:'flex',gap:8}}><input value={desc} onChange={e=>setDesc(e.target.value)} placeholder="Descripción"/><input value={amt} onChange={e=>setAmt(e.target.value)} placeholder="Monto"/><button type="submit">Agregar</button></form>
  <ul>{exp.map((e:any)=><li key={e.id}>{e.description} - {e.amount}</li>)}</ul></div>
}