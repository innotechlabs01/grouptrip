import Cookies from 'js-cookie'
export const getToken = () => Cookies.get('access_token')
export const setToken = (t:string)=> Cookies.set('access_token',t,{secure:true,sameSite:'Lax'})
export const clearToken = ()=> Cookies.remove('access_token')
export const isAuthed = () => !!getToken()
