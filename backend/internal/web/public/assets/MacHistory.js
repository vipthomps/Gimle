import{Y as w,Z as y,_ as v,o as m,O as H,e as f,S as d,t as I,a as D,s as h,f as M,C as b,F as g}from"./index.js";async function u(a,r){let e=[];return r===""?e=await w(a):e=await y(a,r),e!=null?(e.sort((s,t)=>s.Date<t.Date?1:-1),e):[]}var x=I("<i>");function F(a){const[r,e]=v([]);let s;return m(async()=>{const t=await u(a.mac,a.date);e(t),s=setInterval(async()=>{const o=await u(a.mac,a.date);e(o)},6e4)}),H(()=>{clearInterval(s)}),f(g,{each:r,children:(t,o)=>f(d,{get when(){return o()<b()},get children(){var i=x();return D(n=>{var c="Date:"+t.Date+`
Iface:`+t.Iface+`
IP:`+t.IP+`
Known:`+t.Known,l=t.Now===0?"my-box-off":"my-box-on";return c!==n.e&&h(i,"title",n.e=c),l!==n.t&&M(i,n.t=l),n},{e:void 0,t:void 0}),i}})})}export{F as M};
