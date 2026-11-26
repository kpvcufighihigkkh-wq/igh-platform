import{bj as U,b2 as I,b4 as ge,b5 as ve,b0 as _,bk as S,a1 as ze,aT as me,t as N,u as $,w as R,x as E,aA as je,p as Ae,a9 as te,au as d,bl as A,c as Ee,N as Ie,bI as pe,bP as _e,bS as q,bT as be,Q as M,E as De,W as Ne,h as Le,I as ie,ae as He,a2 as Ke,ar as Xe,b8 as Ye,bW as We,D as Ue,aW as qe,b_ as Q,S as Ve,k as Ce,bV as se,aS as Z,bX as V,ay as J,aZ as le,bC as L,a_ as ae,bf as W,an as Ge,aK as Qe,at as Ze,aY as Je}from"./index-DpE57SGd.js";import{a as et,_ as tt,c as nt}from"./Card-B_MZhib7.js";import{b as G,r as re,c as K}from"./resolve-slot-BGAT8vQU.js";import{B as ce}from"./Button-Dnw4VaDu.js";import{b as ot,F as it,c as st,i as ye,m as lt,d as at,p as rt,f as ct,L as dt,z as ut,j as ft}from"./fade-in-scale-up.cssr-D4X5mSnL.js";import{a as ht,u as gt}from"./use-lock-html-scroll-BxyRyhTF.js";import{i as ke,h as we}from"./utils-BMs8-KVU.js";import{e as vt}from"./index-DsSU6HIH.js";const D=S(null);function de(e){if(e.clientX>0||e.clientY>0)D.value={x:e.clientX,y:e.clientY};else{const{target:a}=e;if(a instanceof Element){const{left:i,top:r,width:m,height:g}=a.getBoundingClientRect();i>0||r>0?D.value={x:i+m/2,y:r+g/2}:D.value={x:0,y:0}}else D.value=null}}let X=0,ue=!0;function mt(){if(!ke)return U(S(null));X===0&&I("click",document,de,!0);const e=()=>{X+=1};return ue&&(ue=we())?(ge(e),ve(()=>{X-=1,X===0&&_("click",document,de,!0)})):e(),U(D)}const pt=S(void 0);let Y=0;function fe(){pt.value=Date.now()}let he=!0;function bt(e){if(!ke)return U(S(!1));const a=S(!1);let i=null;function r(){i!==null&&window.clearTimeout(i)}function m(){r(),a.value=!0,i=window.setTimeout(()=>{a.value=!1},e)}Y===0&&I("click",window,fe,!0);const g=()=>{Y+=1,I("click",window,m,!0)};return he&&(he=we())?(ge(g),ve(()=>{Y-=1,Y===0&&_("click",window,fe,!0),_("click",window,m,!0),r()})):g(),U(a)}const Ct=ze("n-dialog-provider"),ne={icon:Function,type:{type:String,default:"default"},title:[String,Function],closable:{type:Boolean,default:!0},negativeText:String,positiveText:String,positiveButtonProps:Object,negativeButtonProps:Object,content:[String,Function],action:Function,showIcon:{type:Boolean,default:!0},loading:Boolean,bordered:Boolean,iconPlacement:String,titleClass:[String,Array],titleStyle:[String,Object],contentClass:[String,Array],contentStyle:[String,Object],actionClass:[String,Array],actionStyle:[String,Object],onPositiveClick:Function,onNegativeClick:Function,onClose:Function,closeFocusable:Boolean},yt=me(ne),kt=N([$("dialog",`
 --n-icon-margin: var(--n-icon-margin-top) var(--n-icon-margin-right) var(--n-icon-margin-bottom) var(--n-icon-margin-left);
 word-break: break-word;
 line-height: var(--n-line-height);
 position: relative;
 background: var(--n-color);
 color: var(--n-text-color);
 box-sizing: border-box;
 margin: auto;
 border-radius: var(--n-border-radius);
 padding: var(--n-padding);
 transition: 
 border-color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 `,[R("icon",`
 color: var(--n-icon-color);
 `),E("bordered",`
 border: var(--n-border);
 `),E("icon-top",[R("close",`
 margin: var(--n-close-margin);
 `),R("icon",`
 margin: var(--n-icon-margin);
 `),R("content",`
 text-align: center;
 `),R("title",`
 justify-content: center;
 `),R("action",`
 justify-content: center;
 `)]),E("icon-left",[R("icon",`
 margin: var(--n-icon-margin);
 `),E("closable",[R("title",`
 padding-right: calc(var(--n-close-size) + 6px);
 `)])]),R("close",`
 position: absolute;
 right: 0;
 top: 0;
 margin: var(--n-close-margin);
 transition:
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 z-index: 1;
 `),R("content",`
 font-size: var(--n-font-size);
 margin: var(--n-content-margin);
 position: relative;
 word-break: break-word;
 `,[E("last","margin-bottom: 0;")]),R("action",`
 display: flex;
 justify-content: flex-end;
 `,[N("> *:not(:last-child)",`
 margin-right: var(--n-action-space);
 `)]),R("icon",`
 font-size: var(--n-icon-size);
 transition: color .3s var(--n-bezier);
 `),R("title",`
 transition: color .3s var(--n-bezier);
 display: flex;
 align-items: center;
 font-size: var(--n-title-font-size);
 font-weight: var(--n-title-font-weight);
 color: var(--n-title-text-color);
 `),$("dialog-icon-container",`
 display: flex;
 justify-content: center;
 `)]),je($("dialog",`
 width: 446px;
 max-width: calc(100vw - 32px);
 `)),$("dialog",[Ae(`
 width: 446px;
 max-width: calc(100vw - 32px);
 `)])]),wt={default:()=>d(ie,null),info:()=>d(ie,null),success:()=>d(Le,null),warning:()=>d(Ne,null),error:()=>d(De,null)},xt=te({name:"Dialog",alias:["NimbusConfirmCard","Confirm"],props:Object.assign(Object.assign({},q.props),ne),slots:Object,setup(e){const{mergedComponentPropsRef:a,mergedClsPrefixRef:i,inlineThemeDisabled:r,mergedRtlRef:m}=pe(e),g=_e("Dialog",m,i),b=M(()=>{var u,f;const{iconPlacement:w}=e;return w||((f=(u=a==null?void 0:a.value)===null||u===void 0?void 0:u.Dialog)===null||f===void 0?void 0:f.iconPlacement)||"left"});function p(u){const{onPositiveClick:f}=e;f&&f(u)}function s(u){const{onNegativeClick:f}=e;f&&f(u)}function x(){const{onClose:u}=e;u&&u()}const k=q("Dialog","-dialog",kt,He,e,i),C=M(()=>{const{type:u}=e,f=b.value,{common:{cubicBezierEaseInOut:w},self:{fontSize:P,lineHeight:y,border:l,titleTextColor:O,textColor:T,color:v,closeBorderRadius:t,closeColorHover:o,closeColorPressed:n,closeIconColor:h,closeIconColorHover:B,closeIconColorPressed:F,closeIconSize:z,borderRadius:j,titleFontWeight:xe,titleFontSize:Pe,padding:Be,iconSize:Re,actionSpace:Se,contentMargin:Oe,closeSize:Te,[f==="top"?"iconMarginIconTop":"iconMargin"]:Fe,[f==="top"?"closeMarginIconTop":"closeMargin"]:Me,[Ke("iconColor",u)]:$e}}=k.value,H=Xe(Fe);return{"--n-font-size":P,"--n-icon-color":$e,"--n-bezier":w,"--n-close-margin":Me,"--n-icon-margin-top":H.top,"--n-icon-margin-right":H.right,"--n-icon-margin-bottom":H.bottom,"--n-icon-margin-left":H.left,"--n-icon-size":Re,"--n-close-size":Te,"--n-close-icon-size":z,"--n-close-border-radius":t,"--n-close-color-hover":o,"--n-close-color-pressed":n,"--n-close-icon-color":h,"--n-close-icon-color-hover":B,"--n-close-icon-color-pressed":F,"--n-color":v,"--n-text-color":T,"--n-border-radius":j,"--n-padding":Be,"--n-line-height":y,"--n-border":l,"--n-content-margin":Oe,"--n-title-font-size":Pe,"--n-title-font-weight":xe,"--n-title-text-color":O,"--n-action-space":Se}}),c=r?be("dialog",M(()=>`${e.type[0]}${b.value[0]}`),C,e):void 0;return{mergedClsPrefix:i,rtlEnabled:g,mergedIconPlacement:b,mergedTheme:k,handlePositiveClick:p,handleNegativeClick:s,handleCloseClick:x,cssVars:r?void 0:C,themeClass:c==null?void 0:c.themeClass,onRender:c==null?void 0:c.onRender}},render(){var e;const{bordered:a,mergedIconPlacement:i,cssVars:r,closable:m,showIcon:g,title:b,content:p,action:s,negativeText:x,positiveText:k,positiveButtonProps:C,negativeButtonProps:c,handlePositiveClick:u,handleNegativeClick:f,mergedTheme:w,loading:P,type:y,mergedClsPrefix:l}=this;(e=this.onRender)===null||e===void 0||e.call(this);const O=g?d(Ee,{clsPrefix:l,class:`${l}-dialog__icon`},{default:()=>G(this.$slots.icon,v=>v||(this.icon?A(this.icon):wt[this.type]()))}):null,T=G(this.$slots.action,v=>v||k||x||s?d("div",{class:[`${l}-dialog__action`,this.actionClass],style:this.actionStyle},v||(s?[A(s)]:[this.negativeText&&d(ce,Object.assign({theme:w.peers.Button,themeOverrides:w.peerOverrides.Button,ghost:!0,size:"small",onClick:f},c),{default:()=>A(this.negativeText)}),this.positiveText&&d(ce,Object.assign({theme:w.peers.Button,themeOverrides:w.peerOverrides.Button,size:"small",type:y==="default"?"primary":y,disabled:P,loading:P,onClick:u},C),{default:()=>A(this.positiveText)})])):null);return d("div",{class:[`${l}-dialog`,this.themeClass,this.closable&&`${l}-dialog--closable`,`${l}-dialog--icon-${i}`,a&&`${l}-dialog--bordered`,this.rtlEnabled&&`${l}-dialog--rtl`],style:r,role:"dialog"},m?G(this.$slots.close,v=>{const t=[`${l}-dialog__close`,this.rtlEnabled&&`${l}-dialog--rtl`];return v?d("div",{class:t},v):d(Ie,{focusable:this.closeFocusable,clsPrefix:l,class:t,onClick:this.handleCloseClick})}):null,g&&i==="top"?d("div",{class:`${l}-dialog-icon-container`},O):null,d("div",{class:[`${l}-dialog__title`,this.titleClass],style:this.titleStyle},g&&i==="left"?O:null,re(this.$slots.header,()=>[A(b)])),d("div",{class:[`${l}-dialog__content`,T?"":`${l}-dialog__content--last`,this.contentClass],style:this.contentStyle},re(this.$slots.default,()=>[A(p)])),T)}}),ee="n-draggable";function Pt(e,a){let i;const r=M(()=>e.value!==!1),m=M(()=>r.value?ee:""),g=M(()=>{const s=e.value;return s===!0||s===!1?!0:s?s.bounds!=="none":!0});function b(s){const x=s.querySelector(`.${ee}`);if(!x||!m.value)return;let k=0,C=0,c=0,u=0,f=0,w=0,P,y=null,l=null;function O(o){o.preventDefault(),P=o;const{x:n,y:h,right:B,bottom:F}=s.getBoundingClientRect();C=n,u=h,k=window.innerWidth-B,c=window.innerHeight-F;const{left:z,top:j}=s.style;f=+j.slice(0,-2),w=+z.slice(0,-2)}function T(){l&&(s.style.top=`${l.y}px`,s.style.left=`${l.x}px`,l=null),y=null}function v(o){if(!P)return;const{clientX:n,clientY:h}=P;let B=o.clientX-n,F=o.clientY-h;g.value&&(B>k?B=k:-B>C&&(B=-C),F>c?F=c:-F>u&&(F=-u));const z=B+w,j=F+f;l={x:z,y:j},y||(y=requestAnimationFrame(T))}function t(){P=void 0,y&&(cancelAnimationFrame(y),y=null),l&&(s.style.top=`${l.y}px`,s.style.left=`${l.x}px`,l=null),a.onEnd(s)}I("mousedown",x,O),I("mousemove",window,v),I("mouseup",window,t),i=()=>{y&&cancelAnimationFrame(y),_("mousedown",x,O),_("mousemove",window,v),_("mouseup",window,t)}}function p(){i&&(i(),i=void 0)}return Ye(p),{stopDrag:p,startDrag:b,draggableRef:r,draggableClassRef:m}}const oe=Object.assign(Object.assign({},et),ne),Bt=me(oe),Rt=te({name:"ModalBody",inheritAttrs:!1,slots:Object,props:Object.assign(Object.assign({show:{type:Boolean,required:!0},preset:String,displayDirective:{type:String,required:!0},trapFocus:{type:Boolean,default:!0},autoFocus:{type:Boolean,default:!0},blockScroll:Boolean,draggable:{type:[Boolean,Object],default:!1},maskHidden:Boolean},oe),{renderMask:Function,onClickoutside:Function,onBeforeLeave:{type:Function,required:!0},onAfterLeave:{type:Function,required:!0},onPositiveClick:{type:Function,required:!0},onNegativeClick:{type:Function,required:!0},onClose:{type:Function,required:!0},onAfterEnter:Function,onEsc:Function}),setup(e){const a=S(null),i=S(null),r=S(e.show),m=S(null),g=S(null),b=J(ye);let p=null;V(L(e,"show"),n=>{n&&(p=b.getMousePosition())},{immediate:!0});const{stopDrag:s,startDrag:x,draggableRef:k,draggableClassRef:C}=Pt(L(e,"draggable"),{onEnd:n=>{w(n)}}),c=M(()=>ae([e.titleClass,C.value])),u=M(()=>ae([e.headerClass,C.value]));V(L(e,"show"),n=>{n&&(r.value=!0)}),ht(M(()=>e.blockScroll&&r.value));function f(){if(b.transformOriginRef.value==="center")return"";const{value:n}=m,{value:h}=g;if(n===null||h===null)return"";if(i.value){const B=i.value.containerScrollTop;return`${n}px ${h+B}px`}return""}function w(n){if(b.transformOriginRef.value==="center"||!p||!i.value)return;const h=i.value.containerScrollTop,{offsetLeft:B,offsetTop:F}=n,z=p.y,j=p.x;m.value=-(B-j),g.value=-(F-z-h),n.style.transformOrigin=f()}function P(n){le(()=>{w(n)})}function y(n){n.style.transformOrigin=f(),e.onBeforeLeave()}function l(n){const h=n;k.value&&x(h),e.onAfterEnter&&e.onAfterEnter(h)}function O(){r.value=!1,m.value=null,g.value=null,s(),e.onAfterLeave()}function T(){const{onClose:n}=e;n&&n()}function v(){e.onNegativeClick()}function t(){e.onPositiveClick()}const o=S(null);return V(o,n=>{n&&le(()=>{const h=n.el;h&&a.value!==h&&(a.value=h)})}),W(lt,a),W(at,null),W(rt,null),{mergedTheme:b.mergedThemeRef,appear:b.appearRef,isMounted:b.isMountedRef,mergedClsPrefix:b.mergedClsPrefixRef,bodyRef:a,scrollbarRef:i,draggableClass:C,displayed:r,childNodeRef:o,cardHeaderClass:u,dialogTitleClass:c,handlePositiveClick:t,handleNegativeClick:v,handleCloseClick:T,handleAfterEnter:l,handleAfterLeave:O,handleBeforeLeave:y,handleEnter:P}},render(){const{$slots:e,$attrs:a,handleEnter:i,handleAfterEnter:r,handleAfterLeave:m,handleBeforeLeave:g,preset:b,mergedClsPrefix:p}=this;let s=null;if(!b){if(s=ot("default",e.default,{draggableClass:this.draggableClass}),!s){We("modal","default slot is empty");return}s=Ue(s),s.props=qe({class:`${p}-modal`},a,s.props||{})}return this.displayDirective==="show"||this.displayed||this.show?Q(d("div",{role:"none",class:[`${p}-modal-body-wrapper`,this.maskHidden&&`${p}-modal-body-wrapper--mask-hidden`]},d(Ve,{ref:"scrollbarRef",theme:this.mergedTheme.peers.Scrollbar,themeOverrides:this.mergedTheme.peerOverrides.Scrollbar,contentClass:`${p}-modal-scroll-content`},{default:()=>{var x;return[(x=this.renderMask)===null||x===void 0?void 0:x.call(this),d(it,{disabled:!this.trapFocus||this.maskHidden,active:this.show,onEsc:this.onEsc,autoFocus:this.autoFocus},{default:()=>{var k;return d(Ce,{name:"fade-in-scale-up-transition",appear:(k=this.appear)!==null&&k!==void 0?k:this.isMounted,onEnter:i,onAfterEnter:r,onAfterLeave:m,onBeforeLeave:g},{default:()=>{const C=[[se,this.show]],{onClickoutside:c}=this;return c&&C.push([st,this.onClickoutside,void 0,{capture:!0}]),Q(this.preset==="confirm"||this.preset==="dialog"?d(xt,Object.assign({},this.$attrs,{class:[`${p}-modal`,this.$attrs.class],ref:"bodyRef",theme:this.mergedTheme.peers.Dialog,themeOverrides:this.mergedTheme.peerOverrides.Dialog},Z(this.$props,yt),{titleClass:this.dialogTitleClass,"aria-modal":"true"}),e):this.preset==="card"?d(tt,Object.assign({},this.$attrs,{ref:"bodyRef",class:[`${p}-modal`,this.$attrs.class],theme:this.mergedTheme.peers.Card,themeOverrides:this.mergedTheme.peerOverrides.Card},Z(this.$props,nt),{headerClass:this.cardHeaderClass,"aria-modal":"true",role:"dialog"}),e):this.childNodeRef=s,C)}})}})]}})),[[se,this.displayDirective==="if"||this.displayed||this.show]]):null}}),St=N([$("modal-container",`
 position: fixed;
 left: 0;
 top: 0;
 height: 0;
 width: 0;
 display: flex;
 `),$("modal-mask",`
 position: fixed;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 background-color: rgba(0, 0, 0, .4);
 `,[Ge({enterDuration:".25s",leaveDuration:".25s",enterCubicBezier:"var(--n-bezier-ease-out)",leaveCubicBezier:"var(--n-bezier-ease-out)"})]),$("modal-body-wrapper",`
 position: fixed;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 overflow: visible;
 `,[$("modal-scroll-content",`
 min-height: 100%;
 display: flex;
 position: relative;
 `),E("mask-hidden","pointer-events: none;",[$("modal-scroll-content",[N("> *",`
 pointer-events: all;
 `)])])]),$("modal",`
 position: relative;
 align-self: center;
 color: var(--n-text-color);
 margin: auto;
 box-shadow: var(--n-box-shadow);
 `,[ct({duration:".25s",enterScale:".5"}),N(`.${ee}`,`
 cursor: move;
 user-select: none;
 `)])]),Ot=Object.assign(Object.assign(Object.assign(Object.assign({},q.props),{show:Boolean,showMask:{type:Boolean,default:!0},maskClosable:{type:Boolean,default:!0},preset:String,to:[String,Object],displayDirective:{type:String,default:"if"},transformOrigin:{type:String,default:"mouse"},zIndex:Number,autoFocus:{type:Boolean,default:!0},trapFocus:{type:Boolean,default:!0},closeOnEsc:{type:Boolean,default:!0},blockScroll:{type:Boolean,default:!0}}),oe),{draggable:[Boolean,Object],onEsc:Function,"onUpdate:show":[Function,Array],onUpdateShow:[Function,Array],onAfterEnter:Function,onBeforeLeave:Function,onAfterLeave:Function,onClose:Function,onPositiveClick:Function,onNegativeClick:Function,onMaskClick:Function,internalDialog:Boolean,internalModal:Boolean,internalAppear:{type:Boolean,default:void 0},overlayStyle:[String,Object],onBeforeHide:Function,onAfterHide:Function,onHide:Function,unstableShowMask:{type:Boolean,default:void 0}}),It=te({name:"Modal",inheritAttrs:!1,props:Ot,slots:Object,setup(e){const a=S(null),{mergedClsPrefixRef:i,namespaceRef:r,inlineThemeDisabled:m}=pe(e),g=q("Modal","-modal",St,Je,e,i),b=bt(64),p=mt(),s=Qe(),x=e.internalDialog?J(Ct,null):null,k=e.internalModal?J(ft,null):null,C=gt();function c(t){const{onUpdateShow:o,"onUpdate:show":n,onHide:h}=e;o&&K(o,t),n&&K(n,t),h&&!t&&h(t)}function u(){const{onClose:t}=e;t?Promise.resolve(t()).then(o=>{o!==!1&&c(!1)}):c(!1)}function f(){const{onPositiveClick:t}=e;t?Promise.resolve(t()).then(o=>{o!==!1&&c(!1)}):c(!1)}function w(){const{onNegativeClick:t}=e;t?Promise.resolve(t()).then(o=>{o!==!1&&c(!1)}):c(!1)}function P(){const{onBeforeLeave:t,onBeforeHide:o}=e;t&&K(t),o&&o()}function y(){const{onAfterLeave:t,onAfterHide:o}=e;t&&K(t),o&&o()}function l(t){var o;const{onMaskClick:n}=e;n&&n(t),e.maskClosable&&!((o=a.value)===null||o===void 0)&&o.contains(Ze(t))&&c(!1)}function O(t){var o;(o=e.onEsc)===null||o===void 0||o.call(e),e.show&&e.closeOnEsc&&vt(t)&&(C.value||c(!1))}W(ye,{getMousePosition:()=>{const t=x||k;if(t){const{clickedRef:o,clickedPositionRef:n}=t;if(o.value&&n.value)return n.value}return b.value?p.value:null},mergedClsPrefixRef:i,mergedThemeRef:g,isMountedRef:s,appearRef:L(e,"internalAppear"),transformOriginRef:L(e,"transformOrigin")});const T=M(()=>{const{common:{cubicBezierEaseOut:t},self:{boxShadow:o,color:n,textColor:h}}=g.value;return{"--n-bezier-ease-out":t,"--n-box-shadow":o,"--n-color":n,"--n-text-color":h}}),v=m?be("theme-class",void 0,T,e):void 0;return{mergedClsPrefix:i,namespace:r,isMounted:s,containerRef:a,presetProps:M(()=>Z(e,Bt)),handleEsc:O,handleAfterLeave:y,handleClickoutside:l,handleBeforeLeave:P,doUpdateShow:c,handleNegativeClick:w,handlePositiveClick:f,handleCloseClick:u,cssVars:m?void 0:T,themeClass:v==null?void 0:v.themeClass,onRender:v==null?void 0:v.onRender}},render(){const{mergedClsPrefix:e}=this;return d(dt,{to:this.to,show:this.show},{default:()=>{var a;(a=this.onRender)===null||a===void 0||a.call(this);const{showMask:i}=this;return Q(d("div",{role:"none",ref:"containerRef",class:[`${e}-modal-container`,this.themeClass,this.namespace],style:this.cssVars},d(Rt,Object.assign({style:this.overlayStyle},this.$attrs,{ref:"bodyWrapper",displayDirective:this.displayDirective,show:this.show,preset:this.preset,autoFocus:this.autoFocus,trapFocus:this.trapFocus,draggable:this.draggable,blockScroll:this.blockScroll,maskHidden:!i},this.presetProps,{onEsc:this.handleEsc,onClose:this.handleCloseClick,onNegativeClick:this.handleNegativeClick,onPositiveClick:this.handlePositiveClick,onBeforeLeave:this.handleBeforeLeave,onAfterEnter:this.onAfterEnter,onAfterLeave:this.handleAfterLeave,onClickoutside:i?void 0:this.handleClickoutside,renderMask:i?()=>{var r;return d(Ce,{name:"fade-in-transition",key:"mask",appear:(r=this.internalAppear)!==null&&r!==void 0?r:this.isMounted},{default:()=>this.show?d("div",{"aria-hidden":!0,ref:"containerRef",class:`${e}-modal-mask`,onClick:this.handleClickoutside}):null})}:void 0}),this.$slots)),[[ut,{zIndex:this.zIndex,enabled:this.show}]])}})}});export{It as _};
