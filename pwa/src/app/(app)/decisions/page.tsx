import { useEffect, useState } from 'react';
import { apiFetch } from '@/lib/api';
export default function DecisionsPage(){
  const [dec,setDec]=useState<any[]>([]);
  useEffect(()=>{ apiFetch('/decisions').then(setDec).catch(()=>{})},[]);
  return <div className="p-6"><h1 className="text-2xl font-bold">Decisiones</h1><ul>{dec.map(d=><li key={d.id}>{d.question}</li>)}</ul></div>
}