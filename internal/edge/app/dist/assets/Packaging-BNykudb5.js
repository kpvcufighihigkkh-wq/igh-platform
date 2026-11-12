import{h as Y}from"./http-BbgdVIJp.js";import{c as ne,f as V}from"./status-DNl5BFw_.js";import{u as ae,_ as se}from"./LoginModal.vue_vue_type_script_setup_true_lang-DZMq-A2T.js";import{_ as le}from"./ModuleAlert.vue_vue_type_script_setup_true_lang-Bb7-CTIR.js";import{_ as ce}from"./PrintLabelModal.vue_vue_type_script_setup_true_lang-2rRIDKbE.js";import{N as de}from"./Tag-BcaeecHW.js";import{a as ue,_ as pe}from"./Tabs-0qMxZd9H.js";import{B as A}from"./Button-CXNqz1kR.js";import{a9 as T,au as t,c as F,Q as C,I as J,W as K,E as Q,h as ee,aZ as ge,t as G,u,x as z,bK as fe,bU as te,bV as he,bg as me,a2 as E,b8 as ye,b9 as ve,$ as X,Y as O,bJ as P,a4 as B,bD as D,a6 as h,b_ as $,_ as H,bm as R,ba as j,Z as U}from"./index-C3RLYDzj.js";import{f as I}from"./get-CGa66pML.js";import{a as be,_ as _e}from"./DescriptionsItem-MQ2hepqk.js";import{N as xe}from"./Space-BsaZ5b9_.js";import{a as $e}from"./DataTable-DZ45AWQ6.js";import{_ as ke}from"./Select-Wv2J1nEa.js";import{_ as Ce}from"./_plugin-vue_export-helper-DlAUqK2U.js";import"./Form-ByiujTiD.js";import"./FormItem-W7Y2FDKg.js";import"./resolve-slot-Bwa-_YNe.js";import"./Input-CHEkeXlw.js";import"./Suffix-CN_cLLcX.js";import"./text-LNsg8-TZ.js";import"./use-compitable-CcAmb2jB.js";import"./Modal-BPRvpg_j.js";import"./Card-BvqAu0ds.js";import"./Popover-DwukWpYw.js";import"./utils-COu6UF-Z.js";import"./get-slot-Gj8iursI.js";import"./use-lock-html-scroll-CINyFu8T.js";import"./is-browser-DqcmxZSF.js";import"./Alert-D2p02qzF.js";import"./InputNumber-DtnzYXN8.js";import"./Spin-CI3N4AOP.js";import"./color-to-class-B0iQgAn2.js";import"./Dropdown-6bGbys2P.js";const we={success:t(ee,null),error:t(Q,null),warning:t(K,null),info:t(J,null)},Se=T({name:"ProgressCircle",props:{clsPrefix:{type:String,required:!0},status:{type:String,required:!0},strokeWidth:{type:Number,required:!0},fillColor:[String,Object],railColor:String,railStyle:[String,Object],percentage:{type:Number,default:0},offsetDegree:{type:Number,default:0},showIndicator:{type:Boolean,required:!0},indicatorTextColor:String,unit:String,viewBoxWidth:{type:Number,required:!0},gapDegree:{type:Number,required:!0},gapOffsetDegree:{type:Number,default:0}},setup(e,{slots:y}){const m=C(()=>{const i="gradient",{fillColor:r}=e;return typeof r=="object"?`${i}-${ge(JSON.stringify(r))}`:i});function v(i,r,n,d){const{gapDegree:b,viewBoxWidth:f,strokeWidth:_}=e,l=50,g=0,c=l,o=0,w=2*l,k=50+_/2,a=`M ${k},${k} m ${g},${c}
      a ${l},${l} 0 1 1 ${o},${-w}
      a ${l},${l} 0 1 1 ${-o},${w}`,s=Math.PI*2*l,x={stroke:d==="rail"?n:typeof e.fillColor=="object"?`url(#${m.value})`:n,strokeDasharray:`${Math.min(i,100)/100*(s-b)}px ${f*8}px`,strokeDashoffset:`-${b/2}px`,transformOrigin:r?"center":void 0,transform:r?`rotate(${r}deg)`:void 0};return{pathString:a,pathStyle:x}}const p=()=>{const i=typeof e.fillColor=="object",r=i?e.fillColor.stops[0]:"",n=i?e.fillColor.stops[1]:"";return i&&t("defs",null,t("linearGradient",{id:m.value,x1:"0%",y1:"100%",x2:"100%",y2:"0%"},t("stop",{offset:"0%","stop-color":r}),t("stop",{offset:"100%","stop-color":n})))};return()=>{const{fillColor:i,railColor:r,strokeWidth:n,offsetDegree:d,status:b,percentage:f,showIndicator:_,indicatorTextColor:l,unit:g,gapOffsetDegree:c,clsPrefix:o}=e,{pathString:w,pathStyle:k}=v(100,0,r,"rail"),{pathString:a,pathStyle:s}=v(f,d,i,"fill"),x=100+n;return t("div",{class:`${o}-progress-content`,role:"none"},t("div",{class:`${o}-progress-graph`,"aria-hidden":!0},t("div",{class:`${o}-progress-graph-circle`,style:{transform:c?`rotate(${c}deg)`:void 0}},t("svg",{viewBox:`0 0 ${x} ${x}`},p(),t("g",null,t("path",{class:`${o}-progress-graph-circle-rail`,d:w,"stroke-width":n,"stroke-linecap":"round",fill:"none",style:k})),t("g",null,t("path",{class:[`${o}-progress-graph-circle-fill`,f===0&&`${o}-progress-graph-circle-fill--empty`],d:a,"stroke-width":n,"stroke-linecap":"round",fill:"none",style:s}))))),_?t("div",null,y.default?t("div",{class:`${o}-progress-custom-content`,role:"none"},y.default()):b!=="default"?t("div",{class:`${o}-progress-icon`,"aria-hidden":!0},t(F,{clsPrefix:o},{default:()=>we[b]})):t("div",{class:`${o}-progress-text`,style:{color:l},role:"none"},t("span",{class:`${o}-progress-text__percentage`},f),t("span",{class:`${o}-progress-text__unit`},g))):null)}}}),ze={success:t(ee,null),error:t(Q,null),warning:t(K,null),info:t(J,null)},Pe=T({name:"ProgressLine",props:{clsPrefix:{type:String,required:!0},percentage:{type:Number,default:0},railColor:String,railStyle:[String,Object],fillColor:[String,Object],status:{type:String,required:!0},indicatorPlacement:{type:String,required:!0},indicatorTextColor:String,unit:{type:String,default:"%"},processing:{type:Boolean,required:!0},showIndicator:{type:Boolean,required:!0},height:[String,Number],railBorderRadius:[String,Number],fillBorderRadius:[String,Number]},setup(e,{slots:y}){const m=C(()=>I(e.height)),v=C(()=>{var r,n;return typeof e.fillColor=="object"?`linear-gradient(to right, ${(r=e.fillColor)===null||r===void 0?void 0:r.stops[0]} , ${(n=e.fillColor)===null||n===void 0?void 0:n.stops[1]})`:e.fillColor}),p=C(()=>e.railBorderRadius!==void 0?I(e.railBorderRadius):e.height!==void 0?I(e.height,{c:.5}):""),i=C(()=>e.fillBorderRadius!==void 0?I(e.fillBorderRadius):e.railBorderRadius!==void 0?I(e.railBorderRadius):e.height!==void 0?I(e.height,{c:.5}):"");return()=>{const{indicatorPlacement:r,railColor:n,railStyle:d,percentage:b,unit:f,indicatorTextColor:_,status:l,showIndicator:g,processing:c,clsPrefix:o}=e;return t("div",{class:`${o}-progress-content`,role:"none"},t("div",{class:`${o}-progress-graph`,"aria-hidden":!0},t("div",{class:[`${o}-progress-graph-line`,{[`${o}-progress-graph-line--indicator-${r}`]:!0}]},t("div",{class:`${o}-progress-graph-line-rail`,style:[{backgroundColor:n,height:m.value,borderRadius:p.value},d]},t("div",{class:[`${o}-progress-graph-line-fill`,c&&`${o}-progress-graph-line-fill--processing`],style:{maxWidth:`${e.percentage}%`,background:v.value,height:m.value,lineHeight:m.value,borderRadius:i.value}},r==="inside"?t("div",{class:`${o}-progress-graph-line-indicator`,style:{color:_}},y.default?y.default():`${b}${f}`):null)))),g&&r==="outside"?t("div",null,y.default?t("div",{class:`${o}-progress-custom-content`,style:{color:_},role:"none"},y.default()):l==="default"?t("div",{role:"none",class:`${o}-progress-icon ${o}-progress-icon--as-text`,style:{color:_}},b,f):t("div",{class:`${o}-progress-icon`,"aria-hidden":!0},t(F,{clsPrefix:o},{default:()=>ze[l]}))):null)}}});function Z(e,y,m=100){return`m ${m/2} ${m/2-e} a ${e} ${e} 0 1 1 0 ${2*e} a ${e} ${e} 0 1 1 0 -${2*e}`}const Be=T({name:"ProgressMultipleCircle",props:{clsPrefix:{type:String,required:!0},viewBoxWidth:{type:Number,required:!0},percentage:{type:Array,default:[0]},strokeWidth:{type:Number,required:!0},circleGap:{type:Number,required:!0},showIndicator:{type:Boolean,required:!0},fillColor:{type:Array,default:()=>[]},railColor:{type:Array,default:()=>[]},railStyle:{type:Array,default:()=>[]}},setup(e,{slots:y}){const m=C(()=>e.percentage.map((i,r)=>`${Math.PI*i/100*(e.viewBoxWidth/2-e.strokeWidth/2*(1+2*r)-e.circleGap*r)*2}, ${e.viewBoxWidth*8}`)),v=(p,i)=>{const r=e.fillColor[i],n=typeof r=="object"?r.stops[0]:"",d=typeof r=="object"?r.stops[1]:"";return typeof e.fillColor[i]=="object"&&t("linearGradient",{id:`gradient-${i}`,x1:"100%",y1:"0%",x2:"0%",y2:"100%"},t("stop",{offset:"0%","stop-color":n}),t("stop",{offset:"100%","stop-color":d}))};return()=>{const{viewBoxWidth:p,strokeWidth:i,circleGap:r,showIndicator:n,fillColor:d,railColor:b,railStyle:f,percentage:_,clsPrefix:l}=e;return t("div",{class:`${l}-progress-content`,role:"none"},t("div",{class:`${l}-progress-graph`,"aria-hidden":!0},t("div",{class:`${l}-progress-graph-circle`},t("svg",{viewBox:`0 0 ${p} ${p}`},t("defs",null,_.map((g,c)=>v(g,c))),_.map((g,c)=>t("g",{key:c},t("path",{class:`${l}-progress-graph-circle-rail`,d:Z(p/2-i/2*(1+2*c)-r*c,i,p),"stroke-width":i,"stroke-linecap":"round",fill:"none",style:[{strokeDashoffset:0,stroke:b[c]},f[c]]}),t("path",{class:[`${l}-progress-graph-circle-fill`,g===0&&`${l}-progress-graph-circle-fill--empty`],d:Z(p/2-i/2*(1+2*c)-r*c,i,p),"stroke-width":i,"stroke-linecap":"round",fill:"none",style:{strokeDasharray:m.value[c],strokeDashoffset:0,stroke:typeof d[c]=="object"?`url(#gradient-${c})`:d[c]}})))))),n&&y.default?t("div",null,t("div",{class:`${l}-progress-text`},y.default())):null)}}}),Ne=G([u("progress",{display:"inline-block"},[u("progress-icon",`
 color: var(--n-icon-color);
 transition: color .3s var(--n-bezier);
 `),z("line",`
 width: 100%;
 display: block;
 `,[u("progress-content",`
 display: flex;
 align-items: center;
 `,[u("progress-graph",{flex:1})]),u("progress-custom-content",{marginLeft:"14px"}),u("progress-icon",`
 width: 30px;
 padding-left: 14px;
 height: var(--n-icon-size-line);
 line-height: var(--n-icon-size-line);
 font-size: var(--n-icon-size-line);
 `,[z("as-text",`
 color: var(--n-text-color-line-outer);
 text-align: center;
 width: 40px;
 font-size: var(--n-font-size);
 padding-left: 4px;
 transition: color .3s var(--n-bezier);
 `)])]),z("circle, dashboard",{width:"120px"},[u("progress-custom-content",`
 position: absolute;
 left: 50%;
 top: 50%;
 transform: translateX(-50%) translateY(-50%);
 display: flex;
 align-items: center;
 justify-content: center;
 `),u("progress-text",`
 position: absolute;
 left: 50%;
 top: 50%;
 transform: translateX(-50%) translateY(-50%);
 display: flex;
 align-items: center;
 color: inherit;
 font-size: var(--n-font-size-circle);
 color: var(--n-text-color-circle);
 font-weight: var(--n-font-weight-circle);
 transition: color .3s var(--n-bezier);
 white-space: nowrap;
 `),u("progress-icon",`
 position: absolute;
 left: 50%;
 top: 50%;
 transform: translateX(-50%) translateY(-50%);
 display: flex;
 align-items: center;
 color: var(--n-icon-color);
 font-size: var(--n-icon-size-circle);
 `)]),z("multiple-circle",`
 width: 200px;
 color: inherit;
 `,[u("progress-text",`
 font-weight: var(--n-font-weight-circle);
 color: var(--n-text-color-circle);
 position: absolute;
 left: 50%;
 top: 50%;
 transform: translateX(-50%) translateY(-50%);
 display: flex;
 align-items: center;
 justify-content: center;
 transition: color .3s var(--n-bezier);
 `)]),u("progress-content",{position:"relative"}),u("progress-graph",{position:"relative"},[u("progress-graph-circle",[G("svg",{verticalAlign:"bottom"}),u("progress-graph-circle-fill",`
 stroke: var(--n-fill-color);
 transition:
 opacity .3s var(--n-bezier),
 stroke .3s var(--n-bezier),
 stroke-dasharray .3s var(--n-bezier);
 `,[z("empty",{opacity:0})]),u("progress-graph-circle-rail",`
 transition: stroke .3s var(--n-bezier);
 overflow: hidden;
 stroke: var(--n-rail-color);
 `)]),u("progress-graph-line",[z("indicator-inside",[u("progress-graph-line-rail",`
 height: 16px;
 line-height: 16px;
 border-radius: 10px;
 `,[u("progress-graph-line-fill",`
 height: inherit;
 border-radius: 10px;
 `),u("progress-graph-line-indicator",`
 background: #0000;
 white-space: nowrap;
 text-align: right;
 margin-left: 14px;
 margin-right: 14px;
 height: inherit;
 font-size: 12px;
 color: var(--n-text-color-line-inner);
 transition: color .3s var(--n-bezier);
 `)])]),z("indicator-inside-label",`
 height: 16px;
 display: flex;
 align-items: center;
 `,[u("progress-graph-line-rail",`
 flex: 1;
 transition: background-color .3s var(--n-bezier);
 `),u("progress-graph-line-indicator",`
 background: var(--n-fill-color);
 font-size: 12px;
 transform: translateZ(0);
 display: flex;
 vertical-align: middle;
 height: 16px;
 line-height: 16px;
 padding: 0 10px;
 border-radius: 10px;
 position: absolute;
 white-space: nowrap;
 color: var(--n-text-color-line-inner);
 transition:
 right .2s var(--n-bezier),
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier);
 `)]),u("progress-graph-line-rail",`
 position: relative;
 overflow: hidden;
 height: var(--n-rail-height);
 border-radius: 5px;
 background-color: var(--n-rail-color);
 transition: background-color .3s var(--n-bezier);
 `,[u("progress-graph-line-fill",`
 background: var(--n-fill-color);
 position: relative;
 border-radius: 5px;
 height: inherit;
 width: 100%;
 max-width: 0%;
 transition:
 background-color .3s var(--n-bezier),
 max-width .2s var(--n-bezier);
 `,[z("processing",[G("&::after",`
 content: "";
 background-image: var(--n-line-bg-processing);
 animation: progress-processing-animation 2s var(--n-bezier) infinite;
 `)])])])])])]),G("@keyframes progress-processing-animation",`
 0% {
 position: absolute;
 left: 0;
 top: 0;
 bottom: 0;
 right: 100%;
 opacity: 1;
 }
 66% {
 position: absolute;
 left: 0;
 top: 0;
 bottom: 0;
 right: 0;
 opacity: 0;
 }
 100% {
 position: absolute;
 left: 0;
 top: 0;
 bottom: 0;
 right: 0;
 opacity: 0;
 }
 `)]),Re=Object.assign(Object.assign({},te.props),{processing:Boolean,type:{type:String,default:"line"},gapDegree:Number,gapOffsetDegree:Number,status:{type:String,default:"default"},railColor:[String,Array],railStyle:[String,Array],color:[String,Array,Object],viewBoxWidth:{type:Number,default:100},strokeWidth:{type:Number,default:7},percentage:[Number,Array],unit:{type:String,default:"%"},showIndicator:{type:Boolean,default:!0},indicatorPosition:{type:String,default:"outside"},indicatorPlacement:{type:String,default:"outside"},indicatorTextColor:String,circleGap:{type:Number,default:1},height:Number,borderRadius:[String,Number],fillBorderRadius:[String,Number],offsetDegree:Number}),De=T({name:"Progress",props:Re,setup(e){const y=C(()=>e.indicatorPlacement||e.indicatorPosition),m=C(()=>{if(e.gapDegree||e.gapDegree===0)return e.gapDegree;if(e.type==="dashboard")return 75}),{mergedClsPrefixRef:v,inlineThemeDisabled:p}=fe(e),i=te("Progress","-progress",Ne,me,e,v),r=C(()=>{const{status:d}=e,{common:{cubicBezierEaseInOut:b},self:{fontSize:f,fontSizeCircle:_,railColor:l,railHeight:g,iconSizeCircle:c,iconSizeLine:o,textColorCircle:w,textColorLineInner:k,textColorLineOuter:a,lineBgProcessing:s,fontWeightCircle:x,[E("iconColor",d)]:S,[E("fillColor",d)]:N}}=i.value;return{"--n-bezier":b,"--n-fill-color":N,"--n-font-size":f,"--n-font-size-circle":_,"--n-font-weight-circle":x,"--n-icon-color":S,"--n-icon-size-circle":c,"--n-icon-size-line":o,"--n-line-bg-processing":s,"--n-rail-color":l,"--n-rail-height":g,"--n-text-color-circle":w,"--n-text-color-line-inner":k,"--n-text-color-line-outer":a}}),n=p?he("progress",C(()=>e.status[0]),r,e):void 0;return{mergedClsPrefix:v,mergedIndicatorPlacement:y,gapDeg:m,cssVars:p?void 0:r,themeClass:n==null?void 0:n.themeClass,onRender:n==null?void 0:n.onRender}},render(){const{type:e,cssVars:y,indicatorTextColor:m,showIndicator:v,status:p,railColor:i,railStyle:r,color:n,percentage:d,viewBoxWidth:b,strokeWidth:f,mergedIndicatorPlacement:_,unit:l,borderRadius:g,fillBorderRadius:c,height:o,processing:w,circleGap:k,mergedClsPrefix:a,gapDeg:s,gapOffsetDegree:x,themeClass:S,$slots:N,onRender:W}=this;return W==null||W(),t("div",{class:[S,`${a}-progress`,`${a}-progress--${e}`,`${a}-progress--${p}`],style:y,"aria-valuemax":100,"aria-valuemin":0,"aria-valuenow":d,role:e==="circle"||e==="line"||e==="dashboard"?"progressbar":"none"},e==="circle"||e==="dashboard"?t(Se,{clsPrefix:a,status:p,showIndicator:v,indicatorTextColor:m,railColor:i,fillColor:n,railStyle:r,offsetDegree:this.offsetDegree,percentage:d,viewBoxWidth:b,strokeWidth:f,gapDegree:s===void 0?e==="dashboard"?75:0:s,gapOffsetDegree:x,unit:l},N):e==="line"?t(Pe,{clsPrefix:a,status:p,showIndicator:v,indicatorTextColor:m,railColor:i,fillColor:n,railStyle:r,percentage:d,processing:w,indicatorPlacement:_,unit:l,fillBorderRadius:c,railBorderRadius:g,height:o},N):e==="multiple-circle"?t(Be,{clsPrefix:a,strokeWidth:f,railColor:i,fillColor:n,railStyle:r,viewBoxWidth:b,percentage:d,showIndicator:v,circleGap:k},N):null)}}),Ie={class:"station"},We={class:"station-header"},qe={class:"header-info"},Oe={key:0,class:"user"},je={class:"clock"},Te={key:0,class:"order-active"},Ae=T({__name:"Packaging",setup(e){const{isLoggedIn:y,userName:m,requireAuth:v,logout:p}=ae(),i=R("orders"),r=R(""),n=R([]),d=R([]),b=R(!1),f=R(!1),_=R({});let l;const g=C(()=>n.value.find(a=>a.status==="in_progress")),c=[{title:"批次",key:"lot_id",width:60},{title:"等级",key:"grade_code",width:60},{title:"目标",key:"target_quantity",width:60},{title:"已完成",key:"packed_quantity",width:70},{title:"状态",key:"status",width:80,render:a=>ne(a.status)},{title:"创建时间",key:"created_at",width:150,render:a=>V(a.created_at)}],o=[{title:"托盘号",key:"code"},{title:"等级",key:"grade_code",width:60},{title:"丝饼数",key:"bobbin_count",width:70},{title:"层数",key:"layers",width:50},{title:"重量(kg)",key:"total_weight",width:80},{title:"时间",key:"created_at",width:150,render:a=>V(a.created_at)}],w=[{title:"托盘号",key:"code"},{title:"等级",key:"grade_code",width:60},{title:"丝饼数",key:"bobbin_count",width:70},{title:"时间",key:"created_at",width:150,render:a=>V(a.created_at)},{title:"操作",key:"a",width:90,render:a=>t(A,{size:"tiny",onClick:()=>v(()=>{_.value={palletCode:a.code,gradeCode:a.grade_code,bobbinCount:a.bobbin_count,layers:a.layers,totalWeight:a.total_weight||0},f.value=!0})},()=>"补打")}];async function k(){var x,S;const[a,s]=await Promise.all([Y.get("/packing-orders?page=1&page_size=20"),Y.get("/pallets?page=1&page_size=20")]);n.value=((x=a.data)==null?void 0:x.items)||[],d.value=((S=s.data)==null?void 0:S.items)||[]}return ye(()=>{l=setInterval(()=>{r.value=new Date().toLocaleTimeString("zh-CN",{hour12:!1}),k()},2e4),r.value=new Date().toLocaleTimeString("zh-CN",{hour12:!1}),k()}),ve(()=>clearInterval(l)),(a,s)=>{const x=de,S=_e,N=be,W=De,re=xe,L=$e,M=pe,oe=ke,ie=ue;return j(),X("div",Ie,[O("header",We,[s[6]||(s[6]=O("h1",null,"包装工位",-1)),O("div",qe,[P(y)()?(j(),X("span",Oe,[B(D(P(m))+" ",1),h(P(A),{text:"",size:"tiny",onClick:P(p)},{default:$(()=>[...s[4]||(s[4]=[B("退出",-1)])]),_:1},8,["onClick"])])):H("",!0),h(x,{type:"success",size:"small"},{default:$(()=>[...s[5]||(s[5]=[B("在线",-1)])]),_:1}),O("span",je,D(r.value),1)])]),h(le),h(ie,{type:"line",value:i.value,"onUpdate:value":s[2]||(s[2]=q=>i.value=q)},{default:$(()=>[h(M,{name:"orders",tab:"订单队列"},{default:$(()=>[g.value?(j(),X("div",Te,[s[7]||(s[7]=O("h3",null,"当前订单",-1)),h(N,{column:3,size:"small",bordered:""},{default:$(()=>[h(S,{label:"批次"},{default:$(()=>[B(D(g.value.lot_id),1)]),_:1}),h(S,{label:"等级"},{default:$(()=>[B(D(g.value.grade_code),1)]),_:1}),h(S,{label:"进度"},{default:$(()=>[B(D(g.value.packed_quantity)+" / "+D(g.value.target_quantity),1)]),_:1})]),_:1}),h(W,{percentage:Math.round(g.value.packed_quantity/g.value.target_quantity*100),type:"line",status:"success",style:{"margin-top":"8px"}},null,8,["percentage"])])):H("",!0),h(re,{style:{margin:"12px 0"}},{default:$(()=>[h(P(A),{size:"small",type:"primary",onClick:s[0]||(s[0]=q=>P(v)(()=>b.value=!0))},{default:$(()=>[...s[8]||(s[8]=[B("新建订单",-1)])]),_:1})]),_:1}),h(L,{columns:c,data:n.value,bordered:!1,size:"small"},null,8,["data"])]),_:1}),h(M,{name:"pallets",tab:"托盘记录"},{default:$(()=>[h(L,{columns:o,data:d.value,bordered:!1,size:"small"},null,8,["data"])]),_:1}),h(M,{name:"print",tab:"打印管理"},{default:$(()=>[h(P(A),{size:"small",onClick:s[1]||(s[1]=q=>P(v)(()=>{f.value=!0}))},{default:$(()=>[...s[9]||(s[9]=[B("补打标签",-1)])]),_:1}),d.value.length===0?(j(),U(oe,{key:0,description:"暂无托盘，包装完成后可补打标签",style:{"margin-top":"24px"}})):(j(),U(L,{key:1,columns:w,data:d.value,bordered:!1,size:"small",style:{"margin-top":"12px"}},null,8,["data"]))]),_:1})]),_:1},8,["value"]),h(se),h(ce,{show:f.value,"onUpdate:show":s[3]||(s[3]=q=>f.value=q),preset:_.value},null,8,["show","preset"])])}}}),_t=Ce(Ae,[["__scopeId","data-v-9e48ecd1"]]);export{_t as default};
