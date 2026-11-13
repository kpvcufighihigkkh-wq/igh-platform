import{ac as F,K as M,P as u,A as v,u as I,w as i,x as y,am as N,t as V,a9 as K,au as t,aW as D,N as Q,e as q,bI as G,bS as $,bP as J,bT as U,Q as R,c as X,E as Y,W as Z,I as oo,h as eo,bk as ro,ar as no,a2 as c}from"./index-Z9hz6zvP.js";import{r as so,b as to}from"./resolve-slot-C44M-6Fz.js";function lo(r){const{lineHeight:o,borderRadius:d,fontWeightStrong:C,baseColor:l,dividerColor:b,actionColor:S,textColor1:g,textColor2:s,closeColorHover:h,closeColorPressed:f,closeIconColor:m,closeIconColorHover:p,closeIconColorPressed:n,infoColor:e,successColor:x,warningColor:z,errorColor:P,fontSize:T}=r;return Object.assign(Object.assign({},M),{fontSize:T,lineHeight:o,titleFontWeight:C,borderRadius:d,border:`1px solid ${b}`,color:S,titleTextColor:g,iconColor:s,contentTextColor:s,closeBorderRadius:d,closeColorHover:h,closeColorPressed:f,closeIconColor:m,closeIconColorHover:p,closeIconColorPressed:n,borderInfo:`1px solid ${u(l,v(e,{alpha:.25}))}`,colorInfo:u(l,v(e,{alpha:.08})),titleTextColorInfo:g,iconColorInfo:e,contentTextColorInfo:s,closeColorHoverInfo:h,closeColorPressedInfo:f,closeIconColorInfo:m,closeIconColorHoverInfo:p,closeIconColorPressedInfo:n,borderSuccess:`1px solid ${u(l,v(x,{alpha:.25}))}`,colorSuccess:u(l,v(x,{alpha:.08})),titleTextColorSuccess:g,iconColorSuccess:x,contentTextColorSuccess:s,closeColorHoverSuccess:h,closeColorPressedSuccess:f,closeIconColorSuccess:m,closeIconColorHoverSuccess:p,closeIconColorPressedSuccess:n,borderWarning:`1px solid ${u(l,v(z,{alpha:.33}))}`,colorWarning:u(l,v(z,{alpha:.08})),titleTextColorWarning:g,iconColorWarning:z,contentTextColorWarning:s,closeColorHoverWarning:h,closeColorPressedWarning:f,closeIconColorWarning:m,closeIconColorHoverWarning:p,closeIconColorPressedWarning:n,borderError:`1px solid ${u(l,v(P,{alpha:.25}))}`,colorError:u(l,v(P,{alpha:.08})),titleTextColorError:g,iconColorError:P,contentTextColorError:s,closeColorHoverError:h,closeColorPressedError:f,closeIconColorError:m,closeIconColorHoverError:p,closeIconColorPressedError:n})}const io={common:F,self:lo},ao=I("alert",`
 line-height: var(--n-line-height);
 border-radius: var(--n-border-radius);
 position: relative;
 transition: background-color .3s var(--n-bezier);
 background-color: var(--n-color);
 text-align: start;
 word-break: break-word;
`,[i("border",`
 border-radius: inherit;
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 transition: border-color .3s var(--n-bezier);
 border: var(--n-border);
 pointer-events: none;
 `),y("closable",[I("alert-body",[i("title",`
 padding-right: 24px;
 `)])]),i("icon",{color:"var(--n-icon-color)"}),I("alert-body",{padding:"var(--n-padding)"},[i("title",{color:"var(--n-title-text-color)"}),i("content",{color:"var(--n-content-text-color)"})]),N({originalTransition:"transform .3s var(--n-bezier)",enterToProps:{transform:"scale(1)"},leaveToProps:{transform:"scale(0.9)"}}),i("icon",`
 position: absolute;
 left: 0;
 top: 0;
 align-items: center;
 justify-content: center;
 display: flex;
 width: var(--n-icon-size);
 height: var(--n-icon-size);
 font-size: var(--n-icon-size);
 margin: var(--n-icon-margin);
 `),i("close",`
 transition:
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier);
 position: absolute;
 right: 0;
 top: 0;
 margin: var(--n-close-margin);
 `),y("show-icon",[I("alert-body",{paddingLeft:"calc(var(--n-icon-margin-left) + var(--n-icon-size) + var(--n-icon-margin-right))"})]),y("right-adjust",[I("alert-body",{paddingRight:"calc(var(--n-close-size) + var(--n-padding) + 2px)"})]),I("alert-body",`
 border-radius: var(--n-border-radius);
 transition: border-color .3s var(--n-bezier);
 `,[i("title",`
 transition: color .3s var(--n-bezier);
 font-size: 16px;
 line-height: 19px;
 font-weight: var(--n-title-font-weight);
 `,[V("& +",[i("content",{marginTop:"9px"})])]),i("content",{transition:"color .3s var(--n-bezier)",fontSize:"var(--n-font-size)"})]),i("icon",{transition:"color .3s var(--n-bezier)"})]),co=Object.assign(Object.assign({},$.props),{title:String,showIcon:{type:Boolean,default:!0},type:{type:String,default:"default"},bordered:{type:Boolean,default:!0},closable:Boolean,onClose:Function,onAfterLeave:Function,onAfterHide:Function}),uo=K({name:"Alert",inheritAttrs:!1,props:co,slots:Object,setup(r){const{mergedClsPrefixRef:o,mergedBorderedRef:d,inlineThemeDisabled:C,mergedRtlRef:l}=G(r),b=$("Alert","-alert",ao,io,r,o),S=J("Alert",l,o),g=R(()=>{const{common:{cubicBezierEaseInOut:n},self:e}=b.value,{fontSize:x,borderRadius:z,titleFontWeight:P,lineHeight:T,iconSize:H,iconMargin:E,iconMarginRtl:W,closeIconSize:_,closeBorderRadius:w,closeSize:A,closeMargin:B,closeMarginRtl:k,padding:j}=e,{type:a}=r,{left:L,right:O}=no(E);return{"--n-bezier":n,"--n-color":e[c("color",a)],"--n-close-icon-size":_,"--n-close-border-radius":w,"--n-close-color-hover":e[c("closeColorHover",a)],"--n-close-color-pressed":e[c("closeColorPressed",a)],"--n-close-icon-color":e[c("closeIconColor",a)],"--n-close-icon-color-hover":e[c("closeIconColorHover",a)],"--n-close-icon-color-pressed":e[c("closeIconColorPressed",a)],"--n-icon-color":e[c("iconColor",a)],"--n-border":e[c("border",a)],"--n-title-text-color":e[c("titleTextColor",a)],"--n-content-text-color":e[c("contentTextColor",a)],"--n-line-height":T,"--n-border-radius":z,"--n-font-size":x,"--n-title-font-weight":P,"--n-icon-size":H,"--n-icon-margin":E,"--n-icon-margin-rtl":W,"--n-close-size":A,"--n-close-margin":B,"--n-close-margin-rtl":k,"--n-padding":j,"--n-icon-margin-left":L,"--n-icon-margin-right":O}}),s=C?U("alert",R(()=>r.type[0]),g,r):void 0,h=ro(!0),f=()=>{const{onAfterLeave:n,onAfterHide:e}=r;n&&n(),e&&e()};return{rtlEnabled:S,mergedClsPrefix:o,mergedBordered:d,visible:h,handleCloseClick:()=>{var n;Promise.resolve((n=r.onClose)===null||n===void 0?void 0:n.call(r)).then(e=>{e!==!1&&(h.value=!1)})},handleAfterLeave:()=>{f()},mergedTheme:b,cssVars:C?void 0:g,themeClass:s==null?void 0:s.themeClass,onRender:s==null?void 0:s.onRender}},render(){var r;return(r=this.onRender)===null||r===void 0||r.call(this),t(q,{onAfterLeave:this.handleAfterLeave},{default:()=>{const{mergedClsPrefix:o,$slots:d}=this,C={class:[`${o}-alert`,this.themeClass,this.closable&&`${o}-alert--closable`,this.showIcon&&`${o}-alert--show-icon`,!this.title&&this.closable&&`${o}-alert--right-adjust`,this.rtlEnabled&&`${o}-alert--rtl`],style:this.cssVars,role:"alert"};return this.visible?t("div",Object.assign({},D(this.$attrs,C)),this.closable&&t(Q,{clsPrefix:o,class:`${o}-alert__close`,onClick:this.handleCloseClick}),this.bordered&&t("div",{class:`${o}-alert__border`}),this.showIcon&&t("div",{class:`${o}-alert__icon`,"aria-hidden":"true"},so(d.icon,()=>[t(X,{clsPrefix:o},{default:()=>{switch(this.type){case"success":return t(eo,null);case"info":return t(oo,null);case"warning":return t(Z,null);case"error":return t(Y,null);default:return null}}})])),t("div",{class:[`${o}-alert-body`,this.mergedBordered&&`${o}-alert-body--bordered`]},to(d.header,l=>{const b=l||this.title;return b?t("div",{class:`${o}-alert-body__title`},b):null}),d.default&&t("div",{class:`${o}-alert-body__content`},d))):null}})}});export{uo as _};
