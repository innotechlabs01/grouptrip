import { useEffect, useState } from 'react';
import { apiFetch } from '@/lib/api';
export default function BudgetPage(){
  const [cats,setCats]=useState<any[]>([]);
  useEffect(()=>{ apiFetch('/budget/categories').then(setCats).catch(()=>{})},[]);
  return <div className="p-6"><h1 className="text-2xl font-bold">Budget</h1><ul>{cats.map(c=><li key={c.id}>{c.name}</li>)}</ul></div>
}