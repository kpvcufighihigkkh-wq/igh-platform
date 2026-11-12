import{t as e,u as a,p as ee,x as s,w as d,aA as oe,aB as re,a9 as ne,au as i,bI as te,bS as $,bP as ae,bT as de,S as le,N as se,Q as z,z as ie,aT as ce,a2 as S,ar as be}from"./index--k3XO6sF.js";import{b as v,e as f,c as ge}from"./resolve-slot-BxqMXCcO.js";const y=a("card-content",`
 flex: 1;
 min-width: 0;
 box-sizing: border-box;
 padding: 0 var(--n-padding-left) var(--n-padding-bottom) var(--n-padding-left);
 font-size: var(--n-font-size);
`),ve=e([a("card",`
 font-size: var(--n-font-size);
 line-height: var(--n-line-height);
 display: flex;
 flex-direction: column;
 width: 100%;
 box-sizing: border-box;
 position: relative;
 border-radius: var(--n-border-radius);
 background-color: var(--n-color);
 color: var(--n-text-color);
 word-break: break-word;
 transition: 
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 `,[ee({background:"var(--n-color-modal)"}),s("hoverable",[e("&:hover","box-shadow: var(--n-box-shadow);")]),s("content-segmented",[e(">",[a("card-content",`
 padding-top: var(--n-padding-bottom);
 `),d("content-scrollbar",[e(">",[a("scrollbar-container",[e(">",[a("card-content",`
 padding-top: var(--n-padding-bottom);
 `)])])])])])]),s("content-soft-segmented",[e(">",[a("card-content",`
 margin: 0 var(--n-padding-left);
 padding: var(--n-padding-bottom) 0;
 `),d("content-scrollbar",[e(">",[a("scrollbar-container",[e(">",[a("card-content",`
 margin: 0 var(--n-padding-left);
 padding: var(--n-padding-bottom) 0;
 `)])])])])])]),s("footer-segmented",[e(">",[d("footer",`
 padding-top: var(--n-padding-bottom);
 `)])]),s("footer-soft-segmented",[e(">",[d("footer",`
 padding: var(--n-padding-bottom) 0;
 margin: 0 var(--n-padding-left);
 `)])]),e(">",[a("card-header",`
 box-sizing: border-box;
 display: flex;
 align-items: center;
 font-size: var(--n-title-font-size);
 padding:
 var(--n-padding-top)
 var(--n-padding-left)
 var(--n-padding-bottom)
 var(--n-padding-left);
 `,[d("main",`
 font-weight: var(--n-title-font-weight);
 transition: color .3s var(--n-bezier);
 flex: 1;
 min-width: 0;
 color: var(--n-title-text-color);
 `),d("extra",`
 display: flex;
 align-items: center;
 font-size: var(--n-font-size);
 font-weight: 400;
 transition: color .3s var(--n-bezier);
 color: var(--n-text-color);
 `),d("close",`
 margin: 0 0 0 8px;
 transition:
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 `)]),d("action",`
 box-sizing: border-box;
 transition:
 background-color .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 background-clip: padding-box;
 background-color: var(--n-action-color);
 `),y,a("card-content",[e("&:first-child",`
 padding-top: var(--n-padding-bottom);
 `)]),d("content-scrollbar",`
 display: flex;
 flex-direction: column;
 `,[e(">",[a("scrollbar-container",[e(">",[y])])]),e("&:first-child >",[a("scrollbar-container",[e(">",[a("card-content",`
 padding-top: var(--n-padding-bottom);
 `)])])])]),d("footer",`
 box-sizing: border-box;
 padding: 0 var(--n-padding-left) var(--n-padding-bottom) var(--n-padding-left);
 font-size: var(--n-font-size);
 `,[e("&:first-child",`
 padding-top: var(--n-padding-bottom);
 `)]),d("action",`
 background-color: var(--n-action-color);
 padding: var(--n-padding-bottom) var(--n-padding-left);
 border-bottom-left-radius: var(--n-border-radius);
 border-bottom-right-radius: var(--n-border-radius);
 `)]),a("card-cover",`
 overflow: hidden;
 width: 100%;
 border-radius: var(--n-border-radius) var(--n-border-radius) 0 0;
 `,[e("img",`
 display: block;
 width: 100%;
 `)]),s("bordered",`
 border: 1px solid var(--n-border-color);
 `,[e("&:target","border-color: var(--n-color-target);")]),s("action-segmented",[e(">",[d("action",[e("&:not(:first-child)",`
 border-top: 1px solid var(--n-border-color);
 `)])])]),s("content-segmented, content-soft-segmented",[e(">",[a("card-content",`
 transition: border-color 0.3s var(--n-bezier);
 `,[e("&:not(:first-child)",`
 border-top: 1px solid var(--n-border-color);
 `)]),d("content-scrollbar",`
 transition: border-color 0.3s var(--n-bezier);
 `,[e("&:not(:first-child)",`
 border-top: 1px solid var(--n-border-color);
 `)])])]),s("footer-segmented, footer-soft-segmented",[e(">",[d("footer",`
 transition: border-color 0.3s var(--n-bezier);
 `,[e("&:not(:first-child)",`
 border-top: 1px solid var(--n-border-color);
 `)])])]),s("embedded",`
 background-color: var(--n-color-embedded);
 `)]),oe(a("card",`
 background: var(--n-color-modal);
 `,[s("embedded",`
 background-color: var(--n-color-embedded-modal);
 `)])),re(a("card",`
 background: var(--n-color-popover);
 `,[s("embedded",`
 background-color: var(--n-color-embedded-popover);
 `)]))]),k={title:[String,Function],contentClass:String,contentStyle:[Object,String],contentScrollable:Boolean,headerClass:String,headerStyle:[Object,String],headerExtraClass:String,headerExtraStyle:[Object,String],footerClass:String,footerStyle:[Object,String],embedded:Boolean,segmented:{type:[Boolean,Object],default:!1},size:String,bordered:{type:Boolean,default:!0},closable:Boolean,hoverable:Boolean,role:String,onClose:[Function,Array],tag:{type:String,default:"div"},cover:Function,content:[String,Function],footer:Function,action:Function,headerExtra:Function,closeFocusable:Boolean},me=ce(k),fe=Object.assign(Object.assign({},$.props),k),ue=ne({name:"Card",props:fe,slots:Object,setup(n){const u=()=>{const{onClose:t}=n;t&&ge(t)},{inlineThemeDisabled:h,mergedClsPrefixRef:r,mergedRtlRef:x,mergedComponentPropsRef:b}=te(n),p=$("Card","-card",ve,ie,n,r),C=ae("Card",x,r),c=z(()=>{var t,g;return n.size||((g=(t=b==null?void 0:b.value)===null||t===void 0?void 0:t.Card)===null||g===void 0?void 0:g.size)||"medium"}),l=z(()=>{const t=c.value,{self:{color:g,colorModal:m,colorTarget:_,textColor:w,titleTextColor:B,titleFontWeight:E,borderColor:P,actionColor:R,borderRadius:F,lineHeight:O,closeIconColor:T,closeIconColorHover:j,closeIconColorPressed:I,closeColorHover:M,closeColorPressed:V,closeBorderRadius:H,closeIconSize:N,closeSize:A,boxShadow:K,colorPopover:L,colorEmbedded:W,colorEmbeddedModal:D,colorEmbeddedPopover:Q,[S("padding",t)]:q,[S("fontSize",t)]:G,[S("titleFontSize",t)]:J},common:{cubicBezierEaseInOut:U}}=p.value,{top:X,left:Y,bottom:Z}=be(q);return{"--n-bezier":U,"--n-border-radius":F,"--n-color":g,"--n-color-modal":m,"--n-color-popover":L,"--n-color-embedded":W,"--n-color-embedded-modal":D,"--n-color-embedded-popover":Q,"--n-color-target":_,"--n-text-color":w,"--n-line-height":O,"--n-action-color":R,"--n-title-text-color":B,"--n-title-font-weight":E,"--n-close-icon-color":T,"--n-close-icon-color-hover":j,"--n-close-icon-color-pressed":I,"--n-close-color-hover":M,"--n-close-color-pressed":V,"--n-border-color":P,"--n-box-shadow":K,"--n-padding-top":X,"--n-padding-bottom":Z,"--n-padding-left":Y,"--n-font-size":G,"--n-title-font-size":J,"--n-close-size":A,"--n-close-icon-size":N,"--n-close-border-radius":H}}),o=h?de("card",z(()=>c.value[0]),l,n):void 0;return{rtlEnabled:C,mergedClsPrefix:r,mergedTheme:p,handleCloseClick:u,cssVars:h?void 0:l,themeClass:o==null?void 0:o.themeClass,onRender:o==null?void 0:o.onRender}},render(){const{segmented:n,bordered:u,hoverable:h,mergedClsPrefix:r,rtlEnabled:x,onRender:b,embedded:p,tag:C,$slots:c}=this;return b==null||b(),i(C,{class:[`${r}-card`,this.themeClass,p&&`${r}-card--embedded`,{[`${r}-card--rtl`]:x,[`${r}-card--content-scrollable`]:this.contentScrollable,[`${r}-card--content${typeof n!="boolean"&&n.content==="soft"?"-soft":""}-segmented`]:n===!0||n!==!1&&n.content,[`${r}-card--footer${typeof n!="boolean"&&n.footer==="soft"?"-soft":""}-segmented`]:n===!0||n!==!1&&n.footer,[`${r}-card--action-segmented`]:n===!0||n!==!1&&n.action,[`${r}-card--bordered`]:u,[`${r}-card--hoverable`]:h}],style:this.cssVars,role:this.role},v(c.cover,l=>{const o=this.cover?f([this.cover()]):l;return o&&i("div",{class:`${r}-card-cover`,role:"none"},o)}),v(c.header,l=>{const{title:o}=this,t=o?f(typeof o=="function"?[o()]:[o]):l;return t||this.closable?i("div",{class:[`${r}-card-header`,this.headerClass],style:this.headerStyle,role:"heading"},i("div",{class:`${r}-card-header__main`,role:"heading"},t),v(c["header-extra"],g=>{const m=this.headerExtra?f([this.headerExtra()]):g;return m&&i("div",{class:[`${r}-card-header__extra`,this.headerExtraClass],style:this.headerExtraStyle},m)}),this.closable&&i(se,{clsPrefix:r,class:`${r}-card-header__close`,onClick:this.handleCloseClick,focusable:this.closeFocusable,absolute:!0})):null}),v(c.default,l=>{const{content:o}=this,t=o?f(typeof o=="function"?[o()]:[o]):l;return t?this.contentScrollable?i(le,{class:`${r}-card__content-scrollbar`,contentClass:[`${r}-card-content`,this.contentClass],contentStyle:this.contentStyle},t):i("div",{class:[`${r}-card-content`,this.contentClass],style:this.contentStyle,role:"none"},t):null}),v(c.footer,l=>{const o=this.footer?f([this.footer()]):l;return o&&i("div",{class:[`${r}-card__footer`,this.footerClass],style:this.footerStyle,role:"none"},o)}),v(c.action,l=>{const o=this.action?f([this.action()]):l;return o&&i("div",{class:`${r}-card__action`,role:"none"},o)}))}});export{ue as _,k as a,me as c};
