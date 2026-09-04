import { useEffect, useState } from 'react';
import { apiFetch } from '@/lib/api';
export default function ExpensesPage(){
  const [exp,setExp]=useState<any[]>([]);
  useEffect(()=>{ apiFetch('/expenses').then(setExp).catch(()=>{})},[]);
  return <div className="p-6"><h1 className="text-2xl font-bold">Gastos</h1><ul>{exp.map(e=><li key={e.id}>{e.description} - {e.amount}</li>)}</ul></div>
}