import{_ as l,d as _,r as d,y as p,b as v,e as m,f as s,t as f,h as a,g,i as w,I as B,o as y}from"./index.a3da76c9.js";/* empty css               *//* empty css              */const b={class:"review"},h={class:"number"},x={class:"bottom"},D=_({__name:"review",setup(I){const e=d(""),o=p().query.code||"000",{getData:n}=v("_","postUserReview"),r=localStorage.getItem("lid")||"",u=()=>{e.value.trim()&&(n({data:{lid:r,msg:e.value}}),e.value="")};return(k,t)=>{const c=B;return y(),m("div",b,[s("div",h,f(a(o)),1),s("div",x,[g(c,{value:a(e),"onUpdate:value":t[0]||(t[0]=i=>w(e)?e.value=i:null),class:"input",placeholder:"\u586B\u5199\u8BC4\u8BBA"},null,8,["value"]),s("span",{class:"send",onClick:u},"\u53D1\u9001")])])}}});var N=l(D,[["__scopeId","data-v-1458b4ae"]]);export{N as default};