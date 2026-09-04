'use client';
import { useEffect, useState } from 'react';
import { API_BASE } from '../../lib/api';
export default function TripsPage(){
  const [trips,setTrips]=useState<any[]>([]);
  useEffect(()=>{ fetch(`${API_BASE}/trips`, {credentials:'include'}).then(r=>r.json()).then(setTrips).catch(()=>setTrips([])); },[]);
  return <main className="p-6"><h1 className="text-2xl font-bold mb-4">Mis viajes</h1><ul>{trips.map((t:any)=><li key={t.id} className="border p-2 mb-2">{t.name}</li>)}</ul></main>
}
