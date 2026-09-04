import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api';
import { useState } from 'react';
export default function BudgetPage(){
  const qc = useQueryClient();
  const { data:cats=[], isLoading:isLoadingCats, isError:eCats, error:eCatsErr } = useQuery({ queryKey:['budget','categories'], queryFn:()=>apiFetch('/budget/categories') });
  const { data:items=[], isLoading:isLoadingItems, isError:eItems, error:eItemsErr } = useQuery({ queryKey:['budget','items'], queryFn:()=>apiFetch('/budget/items') });
  const [name,setName]=useState('');
  const [itemName,setItemName]=useState('');
  const [catId,setCatId]=useState('');
  const [amount,setAmount]=useState('');
  const mutCat = useMutation({ mutationFn:(n:string)=>apiFetch('/budget/categories',{method:'POST',body:JSON.stringify({name:n})}), onSuccess:()=>qc.invalidateQueries({queryKey:['budget','categories']}) });
  const mutItem = useMutation({ mutationFn:()=>apiFetch('/budget/items',{method:'POST',body:JSON.stringify({category_id:catId,name:itemName,amount:parseFloat(amount)})}), onSuccess:()=>qc.invalidateQueries({queryKey:['budget','items']}) });
  const createCat=e=>{e.preventDefault(); if(!name) return; mutCat.mutate(name); setName('');};
  const createItem=e=>{e.preventDefault(); if(!catId||!itemName||!amount) return; mutItem.mutate(); setItemName(''); setAmount('');};
  if(isLoadingCats||isLoadingItems) return <div className="p-6">Cargando...</div>
  if(eCats||eItems) return <div className="p-6 text-red-600">Error: {String(eCats?.error||eItems?.error)}</div>
  return <div className="p-6"><h1 className="text-2xl font-bold">Budget</h1>
  <form onSubmit={createCat} style={{marginBottom:12}}><input value={name} onChange={e=>setName(e.target.value)} placeholder="Nueva categoría"/><button type="submit" disabled={mutCat.isPending}>Crear categoría</button></form>
  <h2>Categorías</h2><ul>{cats.map((c:any)=><li key={c.id}>{c.name}</li>)}</ul>
  <h2>Items</h2>
  <form onSubmit={createItem} style={{marginBottom:12}}>
    <select value={catId} onChange={e=>setCatId(e.target.value)}><option value="">Categoría</option>{cats.map((c:any)=><option key={c.id} value={c.id}>{c.name}</option>)}</select>
    <input value={itemName} onChange={e=>setItemName(e.target.value)} placeholder="Nombre item"/>
    <input value={amount} onChange={e=>setAmount(e.target.value)} placeholder="Monto" type="number"/>
    <button type="submit" disabled={mutItem.isPending}>Crear item</button>
  </form>
  <ul>{items.map((i:any)=><li key={i.id}>{i.name} - ${i.amount}</li>)}</ul>
  </div>
}
