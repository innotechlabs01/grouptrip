export interface Category{ id:string; trip_id:string; name:string }
export interface BudgetItem{ id:string; category_id:string; name:string; amount:number }
export interface Decision{ id:string; trip_id:string; title:string; options:string[] }
export interface Expense{ id:string; trip_id:string; category_id:string; title:string; amount:number }
export interface Trip{ id:string; name:string }
