import{a9 as R,au as d,a5 as Xe,bv as Ze,ac as Je,P as ye,a1 as J,u as h,x as I,S as $e,bI as ne,bS as W,bM as Pe,bT as ie,Q as x,bk as E,bf as X,w as u,t as _,c as Te,ay as G,bC as re,y as Z,am as eo,bl as Q,aT as me,bJ as de,F as oo,aS as ae,e as to,V as ro,aU as no,bY as _e,a0 as io,aW as lo,b9 as M,$ as N,Y as b,a3 as pe,bO as ao,bN as so,b7 as co,br as uo,Z as ho,bZ as U,a6 as K,bH as ze,bB as vo,a4 as Se}from"./index-DpE57SGd.js";import{u as mo}from"./auth-C5CM9VYy.js";import{_ as po}from"./ModuleAlert.vue_vue_type_script_setup_true_lang-Dg3eqDWC.js";import{h as fo}from"./http-ChvKSER2.js";import{C as go,b as xo,N as bo,a as Co}from"./Dropdown-B_xA6tU4.js";import{f as se,u as ue}from"./get-DR01mlWU.js";import{c as L}from"./resolve-slot-BGAT8vQU.js";import{a as wo,e as ce}from"./Popover-t5bRWQKc.js";import{u as ko}from"./use-compitable-CNwyfwXX.js";import{B as yo}from"./Button-Dnw4VaDu.js";import{N as _o}from"./Space-DfuspA4G.js";import{_ as zo}from"./_plugin-vue_export-helper-DlAUqK2U.js";import"./Alert-_3CwTm8t.js";import"./fade-in-scale-up.cssr-D4X5mSnL.js";import"./flatten-BKsH_TtQ.js";import"./utils-BMs8-KVU.js";import"./next-frame-once-C5Ksf8W7.js";import"./is-browser-DqcmxZSF.js";import"./color-to-class-B0iQgAn2.js";import"./get-slot-Bk_rJcZu.js";const So=R({name:"ChevronDownFilled",render(){return d("svg",{viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg"},d("path",{d:"M3.20041 5.73966C3.48226 5.43613 3.95681 5.41856 4.26034 5.70041L8 9.22652L11.7397 5.70041C12.0432 5.41856 12.5177 5.43613 12.7996 5.73966C13.0815 6.0432 13.0639 6.51775 12.7603 6.7996L8.51034 10.7996C8.22258 11.0668 7.77743 11.0668 7.48967 10.7996L3.23966 6.7996C2.93613 6.51775 2.91856 6.0432 3.20041 5.73966Z",fill:"currentColor"}))}});function Io(e){const{baseColor:o,textColor2:t,bodyColor:a,cardColor:l,dividerColor:i,actionColor:v,scrollbarColor:p,scrollbarColorHover:c,invertedColor:s}=e;return{textColor:t,textColorInverted:"#FFF",color:a,colorEmbedded:v,headerColor:l,headerColorInverted:s,footerColor:v,footerColorInverted:s,headerBorderColor:i,headerBorderColorInverted:s,footerBorderColor:i,footerBorderColorInverted:s,siderBorderColor:i,siderBorderColorInverted:s,siderColor:l,siderColorInverted:s,siderToggleButtonBorder:`1px solid ${i}`,siderToggleButtonColor:o,siderToggleButtonIconColor:t,siderToggleButtonIconColorInverted:t,siderToggleBarColor:ye(a,p),siderToggleBarColorHover:ye(a,c),__invertScrollbar:"true"}}const fe=Xe({name:"Layout",common:Je,peers:{Scrollbar:Ze},self:Io}),je=J("n-layout-sider"),ge={type:String,default:"static"},Ro=h("layout",`
 color: var(--n-text-color);
 background-color: var(--n-color);
 box-sizing: border-box;
 position: relative;
 z-index: auto;
 flex: auto;
 overflow: hidden;
 transition:
 box-shadow .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
`,[h("layout-scroll-container",`
 overflow-x: hidden;
 box-sizing: border-box;
 height: 100%;
 `),I("absolute-positioned",`
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 `)]),Ao={embedded:Boolean,position:ge,nativeScrollbar:{type:Boolean,default:!0},scrollbarProps:Object,onScroll:Function,contentClass:String,contentStyle:{type:[String,Object],default:""},hasSider:Boolean,siderPlacement:{type:String,default:"left"}},Oe=J("n-layout");function Me(e){return R({name:e?"LayoutContent":"Layout",props:Object.assign(Object.assign({},W.props),Ao),setup(o){const t=E(null),a=E(null),{mergedClsPrefixRef:l,inlineThemeDisabled:i}=ne(o),v=W("Layout","-layout",Ro,fe,o,l);function p(z,S){if(o.nativeScrollbar){const{value:T}=t;T&&(S===void 0?T.scrollTo(z):T.scrollTo(z,S))}else{const{value:T}=a;T&&T.scrollTo(z,S)}}X(Oe,o);let c=0,s=0;const O=z=>{var S;const T=z.target;c=T.scrollLeft,s=T.scrollTop,(S=o.onScroll)===null||S===void 0||S.call(o,z)};Pe(()=>{if(o.nativeScrollbar){const z=t.value;z&&(z.scrollTop=s,z.scrollLeft=c)}});const j={display:"flex",flexWrap:"nowrap",width:"100%",flexDirection:"row"},f={scrollTo:p},w=x(()=>{const{common:{cubicBezierEaseInOut:z},self:S}=v.value;return{"--n-bezier":z,"--n-color":o.embedded?S.colorEmbedded:S.color,"--n-text-color":S.textColor}}),P=i?ie("layout",x(()=>o.embedded?"e":""),w,o):void 0;return Object.assign({mergedClsPrefix:l,scrollableElRef:t,scrollbarInstRef:a,hasSiderStyle:j,mergedTheme:v,handleNativeElScroll:O,cssVars:i?void 0:w,themeClass:P==null?void 0:P.themeClass,onRender:P==null?void 0:P.onRender},f)},render(){var o;const{mergedClsPrefix:t,hasSider:a}=this;(o=this.onRender)===null||o===void 0||o.call(this);const l=a?this.hasSiderStyle:void 0,i=[this.themeClass,e&&`${t}-layout-content`,`${t}-layout`,`${t}-layout--${this.position}-positioned`];return d("div",{class:i,style:this.cssVars},this.nativeScrollbar?d("div",{ref:"scrollableElRef",class:[`${t}-layout-scroll-container`,this.contentClass],style:[this.contentStyle,l],onScroll:this.handleNativeElScroll},this.$slots):d($e,Object.assign({},this.scrollbarProps,{onScroll:this.onScroll,ref:"scrollbarInstRef",theme:this.mergedTheme.peers.Scrollbar,themeOverrides:this.mergedTheme.peerOverrides.Scrollbar,contentClass:this.contentClass,contentStyle:[this.contentStyle,l]}),this.$slots))}})}const $o=Me(!1),Po=Me(!0),To=h("layout-header",`
 transition:
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 box-sizing: border-box;
 width: 100%;
 background-color: var(--n-color);
 color: var(--n-text-color);
`,[I("absolute-positioned",`
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 `),I("bordered",`
 border-bottom: solid 1px var(--n-border-color);
 `)]),jo={position:ge,inverted:Boolean,bordered:{type:Boolean,default:!1}},Oo=R({name:"LayoutHeader",props:Object.assign(Object.assign({},W.props),jo),setup(e){const{mergedClsPrefixRef:o,inlineThemeDisabled:t}=ne(e),a=W("Layout","-layout-header",To,fe,e,o),l=x(()=>{const{common:{cubicBezierEaseInOut:v},self:p}=a.value,c={"--n-bezier":v};return e.inverted?(c["--n-color"]=p.headerColorInverted,c["--n-text-color"]=p.textColorInverted,c["--n-border-color"]=p.headerBorderColorInverted):(c["--n-color"]=p.headerColor,c["--n-text-color"]=p.textColor,c["--n-border-color"]=p.headerBorderColor),c}),i=t?ie("layout-header",x(()=>e.inverted?"a":"b"),l,e):void 0;return{mergedClsPrefix:o,cssVars:t?void 0:l,themeClass:i==null?void 0:i.themeClass,onRender:i==null?void 0:i.onRender}},render(){var e;const{mergedClsPrefix:o}=this;return(e=this.onRender)===null||e===void 0||e.call(this),d("div",{class:[`${o}-layout-header`,this.themeClass,this.position&&`${o}-layout-header--${this.position}-positioned`,this.bordered&&`${o}-layout-header--bordered`],style:this.cssVars},this.$slots)}}),Mo=h("layout-sider",`
 flex-shrink: 0;
 box-sizing: border-box;
 position: relative;
 z-index: 1;
 color: var(--n-text-color);
 transition:
 color .3s var(--n-bezier),
 border-color .3s var(--n-bezier),
 min-width .3s var(--n-bezier),
 max-width .3s var(--n-bezier),
 transform .3s var(--n-bezier),
 background-color .3s var(--n-bezier);
 background-color: var(--n-color);
 display: flex;
 justify-content: flex-end;
`,[I("bordered",[u("border",`
 content: "";
 position: absolute;
 top: 0;
 bottom: 0;
 width: 1px;
 background-color: var(--n-border-color);
 transition: background-color .3s var(--n-bezier);
 `)]),u("left-placement",[I("bordered",[u("border",`
 right: 0;
 `)])]),I("right-placement",`
 justify-content: flex-start;
 `,[I("bordered",[u("border",`
 left: 0;
 `)]),I("collapsed",[h("layout-toggle-button",[h("base-icon",`
 transform: rotate(180deg);
 `)]),h("layout-toggle-bar",[_("&:hover",[u("top",{transform:"rotate(-12deg) scale(1.15) translateY(-2px)"}),u("bottom",{transform:"rotate(12deg) scale(1.15) translateY(2px)"})])])]),h("layout-toggle-button",`
 left: 0;
 transform: translateX(-50%) translateY(-50%);
 `,[h("base-icon",`
 transform: rotate(0);
 `)]),h("layout-toggle-bar",`
 left: -28px;
 transform: rotate(180deg);
 `,[_("&:hover",[u("top",{transform:"rotate(12deg) scale(1.15) translateY(-2px)"}),u("bottom",{transform:"rotate(-12deg) scale(1.15) translateY(2px)"})])])]),I("collapsed",[h("layout-toggle-bar",[_("&:hover",[u("top",{transform:"rotate(-12deg) scale(1.15) translateY(-2px)"}),u("bottom",{transform:"rotate(12deg) scale(1.15) translateY(2px)"})])]),h("layout-toggle-button",[h("base-icon",`
 transform: rotate(0);
 `)])]),h("layout-toggle-button",`
 transition:
 color .3s var(--n-bezier),
 right .3s var(--n-bezier),
 left .3s var(--n-bezier),
 border-color .3s var(--n-bezier),
 background-color .3s var(--n-bezier);
 cursor: pointer;
 width: 24px;
 height: 24px;
 position: absolute;
 top: 50%;
 right: 0;
 border-radius: 50%;
 display: flex;
 align-items: center;
 justify-content: center;
 font-size: 18px;
 color: var(--n-toggle-button-icon-color);
 border: var(--n-toggle-button-border);
 background-color: var(--n-toggle-button-color);
 box-shadow: 0 2px 4px 0px rgba(0, 0, 0, .06);
 transform: translateX(50%) translateY(-50%);
 z-index: 1;
 `,[h("base-icon",`
 transition: transform .3s var(--n-bezier);
 transform: rotate(180deg);
 `)]),h("layout-toggle-bar",`
 cursor: pointer;
 height: 72px;
 width: 32px;
 position: absolute;
 top: calc(50% - 36px);
 right: -28px;
 `,[u("top, bottom",`
 position: absolute;
 width: 4px;
 border-radius: 2px;
 height: 38px;
 left: 14px;
 transition: 
 background-color .3s var(--n-bezier),
 transform .3s var(--n-bezier);
 `),u("bottom",`
 position: absolute;
 top: 34px;
 `),_("&:hover",[u("top",{transform:"rotate(12deg) scale(1.15) translateY(-2px)"}),u("bottom",{transform:"rotate(-12deg) scale(1.15) translateY(2px)"})]),u("top, bottom",{backgroundColor:"var(--n-toggle-bar-color)"}),_("&:hover",[u("top, bottom",{backgroundColor:"var(--n-toggle-bar-color-hover)"})])]),u("border",`
 position: absolute;
 top: 0;
 right: 0;
 bottom: 0;
 width: 1px;
 transition: background-color .3s var(--n-bezier);
 `),h("layout-sider-scroll-container",`
 flex-grow: 1;
 flex-shrink: 0;
 box-sizing: border-box;
 height: 100%;
 opacity: 0;
 transition: opacity .3s var(--n-bezier);
 max-width: 100%;
 `),I("show-content",[h("layout-sider-scroll-container",{opacity:1})]),I("absolute-positioned",`
 position: absolute;
 left: 0;
 top: 0;
 bottom: 0;
 `)]),No=R({props:{clsPrefix:{type:String,required:!0},onClick:Function},render(){const{clsPrefix:e}=this;return d("div",{onClick:this.onClick,class:`${e}-layout-toggle-bar`},d("div",{class:`${e}-layout-toggle-bar__top`}),d("div",{class:`${e}-layout-toggle-bar__bottom`}))}}),Bo=R({name:"LayoutToggleButton",props:{clsPrefix:{type:String,required:!0},onClick:Function},render(){const{clsPrefix:e}=this;return d("div",{class:`${e}-layout-toggle-button`,onClick:this.onClick},d(Te,{clsPrefix:e},{default:()=>d(go,null)}))}}),Ho={position:ge,bordered:Boolean,collapsedWidth:{type:Number,default:48},width:{type:[Number,String],default:272},contentClass:String,contentStyle:{type:[String,Object],default:""},collapseMode:{type:String,default:"transform"},collapsed:{type:Boolean,default:void 0},defaultCollapsed:Boolean,showCollapsedContent:{type:Boolean,default:!0},showTrigger:{type:[Boolean,String],default:!1},nativeScrollbar:{type:Boolean,default:!0},inverted:Boolean,scrollbarProps:Object,triggerClass:String,triggerStyle:[String,Object],collapsedTriggerClass:String,collapsedTriggerStyle:[String,Object],"onUpdate:collapsed":[Function,Array],onUpdateCollapsed:[Function,Array],onAfterEnter:Function,onAfterLeave:Function,onExpand:[Function,Array],onCollapse:[Function,Array],onScroll:Function},Eo=R({name:"LayoutSider",props:Object.assign(Object.assign({},W.props),Ho),setup(e){const o=G(Oe),t=E(null),a=E(null),l=E(e.defaultCollapsed),i=ue(re(e,"collapsed"),l),v=x(()=>se(i.value?e.collapsedWidth:e.width)),p=x(()=>e.collapseMode!=="transform"?{}:{minWidth:se(e.width)}),c=x(()=>o?o.siderPlacement:"left");function s(A,C){if(e.nativeScrollbar){const{value:y}=t;y&&(C===void 0?y.scrollTo(A):y.scrollTo(A,C))}else{const{value:y}=a;y&&y.scrollTo(A,C)}}function O(){const{"onUpdate:collapsed":A,onUpdateCollapsed:C,onExpand:y,onCollapse:D}=e,{value:V}=i;C&&L(C,!V),A&&L(A,!V),l.value=!V,V?y&&L(y):D&&L(D)}let j=0,f=0;const w=A=>{var C;const y=A.target;j=y.scrollLeft,f=y.scrollTop,(C=e.onScroll)===null||C===void 0||C.call(e,A)};Pe(()=>{if(e.nativeScrollbar){const A=t.value;A&&(A.scrollTop=f,A.scrollLeft=j)}}),X(je,{collapsedRef:i,collapseModeRef:re(e,"collapseMode")});const{mergedClsPrefixRef:P,inlineThemeDisabled:z}=ne(e),S=W("Layout","-layout-sider",Mo,fe,e,P);function T(A){var C,y;A.propertyName==="max-width"&&(i.value?(C=e.onAfterLeave)===null||C===void 0||C.call(e):(y=e.onAfterEnter)===null||y===void 0||y.call(e))}const Y={scrollTo:s},F=x(()=>{const{common:{cubicBezierEaseInOut:A},self:C}=S.value,{siderToggleButtonColor:y,siderToggleButtonBorder:D,siderToggleBarColor:V,siderToggleBarColorHover:le}=C,H={"--n-bezier":A,"--n-toggle-button-color":y,"--n-toggle-button-border":D,"--n-toggle-bar-color":V,"--n-toggle-bar-color-hover":le};return e.inverted?(H["--n-color"]=C.siderColorInverted,H["--n-text-color"]=C.textColorInverted,H["--n-border-color"]=C.siderBorderColorInverted,H["--n-toggle-button-icon-color"]=C.siderToggleButtonIconColorInverted,H.__invertScrollbar=C.__invertScrollbar):(H["--n-color"]=C.siderColor,H["--n-text-color"]=C.textColor,H["--n-border-color"]=C.siderBorderColor,H["--n-toggle-button-icon-color"]=C.siderToggleButtonIconColor),H}),B=z?ie("layout-sider",x(()=>e.inverted?"a":"b"),F,e):void 0;return Object.assign({scrollableElRef:t,scrollbarInstRef:a,mergedClsPrefix:P,mergedTheme:S,styleMaxWidth:v,mergedCollapsed:i,scrollContainerStyle:p,siderPlacement:c,handleNativeElScroll:w,handleTransitionend:T,handleTriggerClick:O,inlineThemeDisabled:z,cssVars:F,themeClass:B==null?void 0:B.themeClass,onRender:B==null?void 0:B.onRender},Y)},render(){var e;const{mergedClsPrefix:o,mergedCollapsed:t,showTrigger:a}=this;return(e=this.onRender)===null||e===void 0||e.call(this),d("aside",{class:[`${o}-layout-sider`,this.themeClass,`${o}-layout-sider--${this.position}-positioned`,`${o}-layout-sider--${this.siderPlacement}-placement`,this.bordered&&`${o}-layout-sider--bordered`,t&&`${o}-layout-sider--collapsed`,(!t||this.showCollapsedContent)&&`${o}-layout-sider--show-content`],onTransitionend:this.handleTransitionend,style:[this.inlineThemeDisabled?void 0:this.cssVars,{maxWidth:this.styleMaxWidth,width:se(this.width)}]},this.nativeScrollbar?d("div",{class:[`${o}-layout-sider-scroll-container`,this.contentClass],onScroll:this.handleNativeElScroll,style:[this.scrollContainerStyle,{overflow:"auto"},this.contentStyle],ref:"scrollableElRef"},this.$slots):d($e,Object.assign({},this.scrollbarProps,{onScroll:this.onScroll,ref:"scrollbarInstRef",style:this.scrollContainerStyle,contentStyle:this.contentStyle,contentClass:this.contentClass,theme:this.mergedTheme.peers.Scrollbar,themeOverrides:this.mergedTheme.peerOverrides.Scrollbar,builtinThemeOverrides:this.inverted&&this.cssVars.__invertScrollbar==="true"?{colorHover:"rgba(255, 255, 255, .4)",color:"rgba(255, 255, 255, .3)"}:void 0}),this.$slots),a?a==="bar"?d(No,{clsPrefix:o,class:t?this.collapsedTriggerClass:this.triggerClass,style:t?this.collapsedTriggerStyle:this.triggerStyle,onClick:this.handleTriggerClick}):d(Bo,{clsPrefix:o,class:t?this.collapsedTriggerClass:this.triggerClass,style:t?this.collapsedTriggerStyle:this.triggerStyle,onClick:this.handleTriggerClick}):null,this.bordered?d("div",{class:`${o}-layout-sider__border`}):null)}}),ee=J("n-menu"),Ne=J("n-submenu"),xe=J("n-menu-item-group"),Ie=[_("&::before","background-color: var(--n-item-color-hover);"),u("arrow",`
 color: var(--n-arrow-color-hover);
 `),u("icon",`
 color: var(--n-item-icon-color-hover);
 `),h("menu-item-content-header",`
 color: var(--n-item-text-color-hover);
 `,[_("a",`
 color: var(--n-item-text-color-hover);
 `),u("extra",`
 color: var(--n-item-text-color-hover);
 `)])],Re=[u("icon",`
 color: var(--n-item-icon-color-hover-horizontal);
 `),h("menu-item-content-header",`
 color: var(--n-item-text-color-hover-horizontal);
 `,[_("a",`
 color: var(--n-item-text-color-hover-horizontal);
 `),u("extra",`
 color: var(--n-item-text-color-hover-horizontal);
 `)])],Lo=_([h("menu",`
 background-color: var(--n-color);
 color: var(--n-item-text-color);
 overflow: hidden;
 transition: background-color .3s var(--n-bezier);
 box-sizing: border-box;
 font-size: var(--n-font-size);
 padding-bottom: 6px;
 `,[I("horizontal",`
 max-width: 100%;
 width: 100%;
 display: flex;
 overflow: hidden;
 padding-bottom: 0;
 `,[h("submenu","margin: 0;"),h("menu-item","margin: 0;"),h("menu-item-content",`
 padding: 0 20px;
 border-bottom: 2px solid #0000;
 `,[_("&::before","display: none;"),I("selected","border-bottom: 2px solid var(--n-border-color-horizontal)")]),h("menu-item-content",[I("selected",[u("icon","color: var(--n-item-icon-color-active-horizontal);"),h("menu-item-content-header",`
 color: var(--n-item-text-color-active-horizontal);
 `,[_("a","color: var(--n-item-text-color-active-horizontal);"),u("extra","color: var(--n-item-text-color-active-horizontal);")])]),I("child-active",`
 border-bottom: 2px solid var(--n-border-color-horizontal);
 `,[h("menu-item-content-header",`
 color: var(--n-item-text-color-child-active-horizontal);
 `,[_("a",`
 color: var(--n-item-text-color-child-active-horizontal);
 `),u("extra",`
 color: var(--n-item-text-color-child-active-horizontal);
 `)]),u("icon",`
 color: var(--n-item-icon-color-child-active-horizontal);
 `)]),Z("disabled",[Z("selected, child-active",[_("&:focus-within",Re)]),I("selected",[q(null,[u("icon","color: var(--n-item-icon-color-active-hover-horizontal);"),h("menu-item-content-header",`
 color: var(--n-item-text-color-active-hover-horizontal);
 `,[_("a","color: var(--n-item-text-color-active-hover-horizontal);"),u("extra","color: var(--n-item-text-color-active-hover-horizontal);")])])]),I("child-active",[q(null,[u("icon","color: var(--n-item-icon-color-child-active-hover-horizontal);"),h("menu-item-content-header",`
 color: var(--n-item-text-color-child-active-hover-horizontal);
 `,[_("a","color: var(--n-item-text-color-child-active-hover-horizontal);"),u("extra","color: var(--n-item-text-color-child-active-hover-horizontal);")])])]),q("border-bottom: 2px solid var(--n-border-color-horizontal);",Re)]),h("menu-item-content-header",[_("a","color: var(--n-item-text-color-horizontal);")])])]),Z("responsive",[h("menu-item-content-header",`
 overflow: hidden;
 text-overflow: ellipsis;
 `)]),I("collapsed",[h("menu-item-content",[I("selected",[_("&::before",`
 background-color: var(--n-item-color-active-collapsed) !important;
 `)]),h("menu-item-content-header","opacity: 0;"),u("arrow","opacity: 0;"),u("icon","color: var(--n-item-icon-color-collapsed);")])]),h("menu-item",`
 height: var(--n-item-height);
 margin-top: 6px;
 position: relative;
 `),h("menu-item-content",`
 box-sizing: border-box;
 line-height: 1.75;
 height: 100%;
 display: grid;
 grid-template-areas: "icon content arrow";
 grid-template-columns: auto 1fr auto;
 align-items: center;
 cursor: pointer;
 position: relative;
 padding-right: 18px;
 transition:
 background-color .3s var(--n-bezier),
 padding-left .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 `,[_("> *","z-index: 1;"),_("&::before",`
 z-index: auto;
 content: "";
 background-color: #0000;
 position: absolute;
 left: 8px;
 right: 8px;
 top: 0;
 bottom: 0;
 pointer-events: none;
 border-radius: var(--n-border-radius);
 transition: background-color .3s var(--n-bezier);
 `),I("disabled",`
 opacity: .45;
 cursor: not-allowed;
 `),I("collapsed",[u("arrow","transform: rotate(0);")]),I("selected",[_("&::before","background-color: var(--n-item-color-active);"),u("arrow","color: var(--n-arrow-color-active);"),u("icon","color: var(--n-item-icon-color-active);"),h("menu-item-content-header",`
 color: var(--n-item-text-color-active);
 `,[_("a","color: var(--n-item-text-color-active);"),u("extra","color: var(--n-item-text-color-active);")])]),I("child-active",[h("menu-item-content-header",`
 color: var(--n-item-text-color-child-active);
 `,[_("a",`
 color: var(--n-item-text-color-child-active);
 `),u("extra",`
 color: var(--n-item-text-color-child-active);
 `)]),u("arrow",`
 color: var(--n-arrow-color-child-active);
 `),u("icon",`
 color: var(--n-item-icon-color-child-active);
 `)]),Z("disabled",[Z("selected, child-active",[_("&:focus-within",Ie)]),I("selected",[q(null,[u("arrow","color: var(--n-arrow-color-active-hover);"),u("icon","color: var(--n-item-icon-color-active-hover);"),h("menu-item-content-header",`
 color: var(--n-item-text-color-active-hover);
 `,[_("a","color: var(--n-item-text-color-active-hover);"),u("extra","color: var(--n-item-text-color-active-hover);")])])]),I("child-active",[q(null,[u("arrow","color: var(--n-arrow-color-child-active-hover);"),u("icon","color: var(--n-item-icon-color-child-active-hover);"),h("menu-item-content-header",`
 color: var(--n-item-text-color-child-active-hover);
 `,[_("a","color: var(--n-item-text-color-child-active-hover);"),u("extra","color: var(--n-item-text-color-child-active-hover);")])])]),I("selected",[q(null,[_("&::before","background-color: var(--n-item-color-active-hover);")])]),q(null,Ie)]),u("icon",`
 grid-area: icon;
 color: var(--n-item-icon-color);
 transition:
 color .3s var(--n-bezier),
 font-size .3s var(--n-bezier),
 margin-right .3s var(--n-bezier);
 box-sizing: content-box;
 display: inline-flex;
 align-items: center;
 justify-content: center;
 `),u("arrow",`
 grid-area: arrow;
 font-size: 16px;
 color: var(--n-arrow-color);
 transform: rotate(180deg);
 opacity: 1;
 transition:
 color .3s var(--n-bezier),
 transform 0.2s var(--n-bezier),
 opacity 0.2s var(--n-bezier);
 `),h("menu-item-content-header",`
 grid-area: content;
 transition:
 color .3s var(--n-bezier),
 opacity .3s var(--n-bezier);
 opacity: 1;
 white-space: nowrap;
 color: var(--n-item-text-color);
 `,[_("a",`
 outline: none;
 text-decoration: none;
 transition: color .3s var(--n-bezier);
 color: var(--n-item-text-color);
 `,[_("&::before",`
 content: "";
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 `)]),u("extra",`
 font-size: .93em;
 color: var(--n-group-text-color);
 transition: color .3s var(--n-bezier);
 `)])]),h("submenu",`
 cursor: pointer;
 position: relative;
 margin-top: 6px;
 `,[h("menu-item-content",`
 height: var(--n-item-height);
 `),h("submenu-children",`
 overflow: hidden;
 padding: 0;
 `,[eo({duration:".2s"})])]),h("menu-item-group",[h("menu-item-group-title",`
 margin-top: 6px;
 color: var(--n-group-text-color);
 cursor: default;
 font-size: .93em;
 height: 36px;
 display: flex;
 align-items: center;
 transition:
 padding-left .3s var(--n-bezier),
 color .3s var(--n-bezier);
 `)])]),h("menu-tooltip",[_("a",`
 color: inherit;
 text-decoration: none;
 `)]),h("menu-divider",`
 transition: background-color .3s var(--n-bezier);
 background-color: var(--n-divider-color);
 height: 1px;
 margin: 6px 18px;
 `)]);function q(e,o){return[I("hover",e,o),_("&:hover",e,o)]}const Be=R({name:"MenuOptionContent",props:{collapsed:Boolean,disabled:Boolean,title:[String,Function],icon:Function,extra:[String,Function],showArrow:Boolean,childActive:Boolean,hover:Boolean,paddingLeft:Number,selected:Boolean,maxIconSize:{type:Number,required:!0},activeIconSize:{type:Number,required:!0},iconMarginRight:{type:Number,required:!0},clsPrefix:{type:String,required:!0},onClick:Function,tmNode:{type:Object,required:!0},isEllipsisPlaceholder:Boolean},setup(e){const{props:o}=G(ee);return{menuProps:o,style:x(()=>{const{paddingLeft:t}=e;return{paddingLeft:t&&`${t}px`}}),iconStyle:x(()=>{const{maxIconSize:t,activeIconSize:a,iconMarginRight:l}=e;return{width:`${t}px`,height:`${t}px`,fontSize:`${a}px`,marginRight:`${l}px`}})}},render(){const{clsPrefix:e,tmNode:o,menuProps:{renderIcon:t,renderLabel:a,renderExtra:l,expandIcon:i}}=this,v=t?t(o.rawNode):Q(this.icon);return d("div",{onClick:p=>{var c;(c=this.onClick)===null||c===void 0||c.call(this,p)},role:"none",class:[`${e}-menu-item-content`,{[`${e}-menu-item-content--selected`]:this.selected,[`${e}-menu-item-content--collapsed`]:this.collapsed,[`${e}-menu-item-content--child-active`]:this.childActive,[`${e}-menu-item-content--disabled`]:this.disabled,[`${e}-menu-item-content--hover`]:this.hover}],style:this.style},v&&d("div",{class:`${e}-menu-item-content__icon`,style:this.iconStyle,role:"none"},[v]),d("div",{class:`${e}-menu-item-content-header`,role:"none"},this.isEllipsisPlaceholder?this.title:a?a(o.rawNode):Q(this.title),this.extra||l?d("span",{class:`${e}-menu-item-content-header__extra`}," ",l?l(o.rawNode):Q(this.extra)):null),this.showArrow?d(Te,{ariaHidden:!0,class:`${e}-menu-item-content__arrow`,clsPrefix:e},{default:()=>i?i(o.rawNode):d(So,null)}):null)}}),te=8;function be(e){const o=G(ee),{props:t,mergedCollapsedRef:a}=o,l=G(Ne,null),i=G(xe,null),v=x(()=>t.mode==="horizontal"),p=x(()=>v.value?t.dropdownPlacement:"tmNodes"in e?"right-start":"right"),c=x(()=>{var f;return Math.max((f=t.collapsedIconSize)!==null&&f!==void 0?f:t.iconSize,t.iconSize)}),s=x(()=>{var f;return!v.value&&e.root&&a.value&&(f=t.collapsedIconSize)!==null&&f!==void 0?f:t.iconSize}),O=x(()=>{if(v.value)return;const{collapsedWidth:f,indent:w,rootIndent:P}=t,{root:z,isGroup:S}=e,T=P===void 0?w:P;return z?a.value?f/2-c.value/2:T:i&&typeof i.paddingLeftRef.value=="number"?w/2+i.paddingLeftRef.value:l&&typeof l.paddingLeftRef.value=="number"?(S?w/2:w)+l.paddingLeftRef.value:0}),j=x(()=>{const{collapsedWidth:f,indent:w,rootIndent:P}=t,{value:z}=c,{root:S}=e;return v.value||!S||!a.value?te:(P===void 0?w:P)+z+te-(f+z)/2});return{dropdownPlacement:p,activeIconSize:s,maxIconSize:c,paddingLeft:O,iconMarginRight:j,NMenu:o,NSubmenu:l,NMenuOptionGroup:i}}const Ce={internalKey:{type:[String,Number],required:!0},root:Boolean,isGroup:Boolean,level:{type:Number,required:!0},title:[String,Function],extra:[String,Function]},Fo=R({name:"MenuDivider",setup(){const e=G(ee),{mergedClsPrefixRef:o,isHorizontalRef:t}=e;return()=>t.value?null:d("div",{class:`${o.value}-menu-divider`})}}),He=Object.assign(Object.assign({},Ce),{tmNode:{type:Object,required:!0},disabled:Boolean,icon:Function,onClick:Function}),Ko=me(He),Vo=R({name:"MenuOption",props:He,setup(e){const o=be(e),{NSubmenu:t,NMenu:a,NMenuOptionGroup:l}=o,{props:i,mergedClsPrefixRef:v,mergedCollapsedRef:p}=a,c=t?t.mergedDisabledRef:l?l.mergedDisabledRef:{value:!1},s=x(()=>c.value||e.disabled);function O(f){const{onClick:w}=e;w&&w(f)}function j(f){s.value||(a.doSelect(e.internalKey,e.tmNode.rawNode),O(f))}return{mergedClsPrefix:v,dropdownPlacement:o.dropdownPlacement,paddingLeft:o.paddingLeft,iconMarginRight:o.iconMarginRight,maxIconSize:o.maxIconSize,activeIconSize:o.activeIconSize,mergedTheme:a.mergedThemeRef,menuProps:i,dropdownEnabled:de(()=>e.root&&p.value&&i.mode!=="horizontal"&&!s.value),selected:de(()=>a.mergedValueRef.value===e.internalKey),mergedDisabled:s,handleClick:j}},render(){const{mergedClsPrefix:e,mergedTheme:o,tmNode:t,menuProps:{renderLabel:a,nodeProps:l}}=this,i=l==null?void 0:l(t.rawNode);return d("div",Object.assign({},i,{role:"menuitem",class:[`${e}-menu-item`,i==null?void 0:i.class]}),d(xo,{theme:o.peers.Tooltip,themeOverrides:o.peerOverrides.Tooltip,trigger:"hover",placement:this.dropdownPlacement,disabled:!this.dropdownEnabled||this.title===void 0,internalExtraClass:["menu-tooltip"]},{default:()=>a?a(t.rawNode):Q(this.title),trigger:()=>d(Be,{tmNode:t,clsPrefix:e,paddingLeft:this.paddingLeft,iconMarginRight:this.iconMarginRight,maxIconSize:this.maxIconSize,activeIconSize:this.activeIconSize,selected:this.selected,title:this.title,extra:this.extra,disabled:this.mergedDisabled,icon:this.icon,onClick:this.handleClick})}))}}),Ee=Object.assign(Object.assign({},Ce),{tmNode:{type:Object,required:!0},tmNodes:{type:Array,required:!0}}),Do=me(Ee),Uo=R({name:"MenuOptionGroup",props:Ee,setup(e){const o=be(e),{NSubmenu:t}=o,a=x(()=>t!=null&&t.mergedDisabledRef.value?!0:e.tmNode.disabled);X(xe,{paddingLeftRef:o.paddingLeft,mergedDisabledRef:a});const{mergedClsPrefixRef:l,props:i}=G(ee);return function(){const{value:v}=l,p=o.paddingLeft.value,{nodeProps:c}=i,s=c==null?void 0:c(e.tmNode.rawNode);return d("div",{class:`${v}-menu-item-group`,role:"group"},d("div",Object.assign({},s,{class:[`${v}-menu-item-group-title`,s==null?void 0:s.class],style:[(s==null?void 0:s.style)||"",p!==void 0?`padding-left: ${p}px;`:""]}),Q(e.title),e.extra?d(oo,null," ",Q(e.extra)):null),d("div",null,e.tmNodes.map(O=>we(O,i))))}}});function he(e){return e.type==="divider"||e.type==="render"}function Go(e){return e.type==="divider"}function we(e,o){const{rawNode:t}=e,{show:a}=t;if(a===!1)return null;if(he(t))return Go(t)?d(Fo,Object.assign({key:e.key},t.props)):null;const{labelField:l}=o,{key:i,level:v,isGroup:p}=e,c=Object.assign(Object.assign({},t),{title:t.title||t[l],extra:t.titleExtra||t.extra,key:i,internalKey:i,level:v,root:v===0,isGroup:p});return e.children?e.isGroup?d(Uo,ae(c,Do,{tmNode:e,tmNodes:e.children,key:i})):d(ve,ae(c,Wo,{key:i,rawNodes:t[o.childrenField],tmNodes:e.children,tmNode:e})):d(Vo,ae(c,Ko,{key:i,tmNode:e}))}const Le=Object.assign(Object.assign({},Ce),{rawNodes:{type:Array,default:()=>[]},tmNodes:{type:Array,default:()=>[]},tmNode:{type:Object,required:!0},disabled:Boolean,icon:Function,onClick:Function,domId:String,virtualChildActive:{type:Boolean,default:void 0},isEllipsisPlaceholder:Boolean}),Wo=me(Le),ve=R({name:"Submenu",props:Le,setup(e){const o=be(e),{NMenu:t,NSubmenu:a}=o,{props:l,mergedCollapsedRef:i,mergedThemeRef:v}=t,p=x(()=>{const{disabled:f}=e;return a!=null&&a.mergedDisabledRef.value||l.disabled?!0:f}),c=E(!1);X(Ne,{paddingLeftRef:o.paddingLeft,mergedDisabledRef:p}),X(xe,null);function s(){const{onClick:f}=e;f&&f()}function O(){p.value||(i.value||t.toggleExpand(e.internalKey),s())}function j(f){c.value=f}return{menuProps:l,mergedTheme:v,doSelect:t.doSelect,inverted:t.invertedRef,isHorizontal:t.isHorizontalRef,mergedClsPrefix:t.mergedClsPrefixRef,maxIconSize:o.maxIconSize,activeIconSize:o.activeIconSize,iconMarginRight:o.iconMarginRight,dropdownPlacement:o.dropdownPlacement,dropdownShow:c,paddingLeft:o.paddingLeft,mergedDisabled:p,mergedValue:t.mergedValueRef,childActive:de(()=>{var f;return(f=e.virtualChildActive)!==null&&f!==void 0?f:t.activePathRef.value.includes(e.internalKey)}),collapsed:x(()=>l.mode==="horizontal"?!1:i.value?!0:!t.mergedExpandedKeysRef.value.includes(e.internalKey)),dropdownEnabled:x(()=>!p.value&&(l.mode==="horizontal"||i.value)),handlePopoverShowChange:j,handleClick:O}},render(){var e;const{mergedClsPrefix:o,menuProps:{renderIcon:t,renderLabel:a}}=this,l=()=>{const{isHorizontal:v,paddingLeft:p,collapsed:c,mergedDisabled:s,maxIconSize:O,activeIconSize:j,title:f,childActive:w,icon:P,handleClick:z,menuProps:{nodeProps:S},dropdownShow:T,iconMarginRight:Y,tmNode:F,mergedClsPrefix:B,isEllipsisPlaceholder:A,extra:C}=this,y=S==null?void 0:S(F.rawNode);return d("div",Object.assign({},y,{class:[`${B}-menu-item`,y==null?void 0:y.class],role:"menuitem"}),d(Be,{tmNode:F,paddingLeft:p,collapsed:c,disabled:s,iconMarginRight:Y,maxIconSize:O,activeIconSize:j,title:f,extra:C,showArrow:!v,childActive:w,clsPrefix:B,icon:P,hover:T,onClick:z,isEllipsisPlaceholder:A}))},i=()=>d(to,null,{default:()=>{const{tmNodes:v,collapsed:p}=this;return p?null:d("div",{class:`${o}-submenu-children`,role:"menu"},v.map(c=>we(c,this.menuProps)))}});return this.root?d(bo,Object.assign({size:"large",trigger:"hover"},(e=this.menuProps)===null||e===void 0?void 0:e.dropdownProps,{themeOverrides:this.mergedTheme.peerOverrides.Dropdown,theme:this.mergedTheme.peers.Dropdown,builtinThemeOverrides:{fontSizeLarge:"14px",optionIconSizeLarge:"18px"},value:this.mergedValue,disabled:!this.dropdownEnabled,placement:this.dropdownPlacement,keyField:this.menuProps.keyField,labelField:this.menuProps.labelField,childrenField:this.menuProps.childrenField,onUpdateShow:this.handlePopoverShowChange,options:this.rawNodes,onSelect:this.doSelect,inverted:this.inverted,renderIcon:t,renderLabel:a}),{default:()=>d("div",{class:`${o}-submenu`,role:"menu","aria-expanded":!this.collapsed,id:this.domId},l(),this.isHorizontal?null:i())}):d("div",{class:`${o}-submenu`,role:"menu","aria-expanded":!this.collapsed,id:this.domId},l(),i())}}),Yo=Object.assign(Object.assign({},W.props),{options:{type:Array,default:()=>[]},collapsed:{type:Boolean,default:void 0},collapsedWidth:{type:Number,default:48},iconSize:{type:Number,default:20},collapsedIconSize:{type:Number,default:24},rootIndent:Number,indent:{type:Number,default:32},labelField:{type:String,default:"label"},keyField:{type:String,default:"key"},childrenField:{type:String,default:"children"},disabledField:{type:String,default:"disabled"},defaultExpandAll:Boolean,defaultExpandedKeys:Array,expandedKeys:Array,value:[String,Number],defaultValue:{type:[String,Number],default:null},mode:{type:String,default:"vertical"},watchProps:{type:Array,default:void 0},disabled:Boolean,show:{type:Boolean,default:!0},inverted:Boolean,"onUpdate:expandedKeys":[Function,Array],onUpdateExpandedKeys:[Function,Array],onUpdateValue:[Function,Array],"onUpdate:value":[Function,Array],expandIcon:Function,renderIcon:Function,renderLabel:Function,renderExtra:Function,dropdownProps:Object,accordion:Boolean,nodeProps:Function,dropdownPlacement:{type:String,default:"bottom"},responsive:Boolean,items:Array,onOpenNamesChange:[Function,Array],onSelect:[Function,Array],onExpandedNamesChange:[Function,Array],expandedNames:Array,defaultExpandedNames:Array}),qo=R({name:"Menu",inheritAttrs:!1,props:Yo,setup(e){const{mergedClsPrefixRef:o,inlineThemeDisabled:t}=ne(e),a=W("Menu","-menu",Lo,no,e,o),l=G(je,null),i=x(()=>{var m;const{collapsed:k}=e;if(k!==void 0)return k;if(l){const{collapseModeRef:r,collapsedRef:g}=l;if(r.value==="width")return(m=g.value)!==null&&m!==void 0?m:!1}return!1}),v=x(()=>{const{keyField:m,childrenField:k,disabledField:r}=e;return ce(e.items||e.options,{getIgnored(g){return he(g)},getChildren(g){return g[k]},getDisabled(g){return g[r]},getKey(g){var $;return($=g[m])!==null&&$!==void 0?$:g.name}})}),p=x(()=>new Set(v.value.treeNodes.map(m=>m.key))),{watchProps:c}=e,s=E(null);c!=null&&c.includes("defaultValue")?_e(()=>{s.value=e.defaultValue}):s.value=e.defaultValue;const O=re(e,"value"),j=ue(O,s),f=E([]),w=()=>{f.value=e.defaultExpandAll?v.value.getNonLeafKeys():e.defaultExpandedNames||e.defaultExpandedKeys||v.value.getPath(j.value,{includeSelf:!1}).keyPath};c!=null&&c.includes("defaultExpandedKeys")?_e(w):w();const P=ko(e,["expandedNames","expandedKeys"]),z=ue(P,f),S=x(()=>v.value.treeNodes),T=x(()=>v.value.getPath(j.value).keyPath);X(ee,{props:e,mergedCollapsedRef:i,mergedThemeRef:a,mergedValueRef:j,mergedExpandedKeysRef:z,activePathRef:T,mergedClsPrefixRef:o,isHorizontalRef:x(()=>e.mode==="horizontal"),invertedRef:re(e,"inverted"),doSelect:Y,toggleExpand:B});function Y(m,k){const{"onUpdate:value":r,onUpdateValue:g,onSelect:$}=e;g&&L(g,m,k),r&&L(r,m,k),$&&L($,m,k),s.value=m}function F(m){const{"onUpdate:expandedKeys":k,onUpdateExpandedKeys:r,onExpandedNamesChange:g,onOpenNamesChange:$}=e;k&&L(k,m),r&&L(r,m),g&&L(g,m),$&&L($,m),f.value=m}function B(m){const k=Array.from(z.value),r=k.findIndex(g=>g===m);if(~r)k.splice(r,1);else{if(e.accordion&&p.value.has(m)){const g=k.findIndex($=>p.value.has($));g>-1&&k.splice(g,1)}k.push(m)}F(k)}const A=m=>{const k=v.value.getPath(m??j.value,{includeSelf:!1}).keyPath;if(!k.length)return;const r=Array.from(z.value),g=new Set([...r,...k]);e.accordion&&p.value.forEach($=>{g.has($)&&!k.includes($)&&g.delete($)}),F(Array.from(g))},C=x(()=>{const{inverted:m}=e,{common:{cubicBezierEaseInOut:k},self:r}=a.value,{borderRadius:g,borderColorHorizontal:$,fontSize:Ye,itemHeight:qe,dividerColor:Qe}=r,n={"--n-divider-color":Qe,"--n-bezier":k,"--n-font-size":Ye,"--n-border-color-horizontal":$,"--n-border-radius":g,"--n-item-height":qe};return m?(n["--n-group-text-color"]=r.groupTextColorInverted,n["--n-color"]=r.colorInverted,n["--n-item-text-color"]=r.itemTextColorInverted,n["--n-item-text-color-hover"]=r.itemTextColorHoverInverted,n["--n-item-text-color-active"]=r.itemTextColorActiveInverted,n["--n-item-text-color-child-active"]=r.itemTextColorChildActiveInverted,n["--n-item-text-color-child-active-hover"]=r.itemTextColorChildActiveInverted,n["--n-item-text-color-active-hover"]=r.itemTextColorActiveHoverInverted,n["--n-item-icon-color"]=r.itemIconColorInverted,n["--n-item-icon-color-hover"]=r.itemIconColorHoverInverted,n["--n-item-icon-color-active"]=r.itemIconColorActiveInverted,n["--n-item-icon-color-active-hover"]=r.itemIconColorActiveHoverInverted,n["--n-item-icon-color-child-active"]=r.itemIconColorChildActiveInverted,n["--n-item-icon-color-child-active-hover"]=r.itemIconColorChildActiveHoverInverted,n["--n-item-icon-color-collapsed"]=r.itemIconColorCollapsedInverted,n["--n-item-text-color-horizontal"]=r.itemTextColorHorizontalInverted,n["--n-item-text-color-hover-horizontal"]=r.itemTextColorHoverHorizontalInverted,n["--n-item-text-color-active-horizontal"]=r.itemTextColorActiveHorizontalInverted,n["--n-item-text-color-child-active-horizontal"]=r.itemTextColorChildActiveHorizontalInverted,n["--n-item-text-color-child-active-hover-horizontal"]=r.itemTextColorChildActiveHoverHorizontalInverted,n["--n-item-text-color-active-hover-horizontal"]=r.itemTextColorActiveHoverHorizontalInverted,n["--n-item-icon-color-horizontal"]=r.itemIconColorHorizontalInverted,n["--n-item-icon-color-hover-horizontal"]=r.itemIconColorHoverHorizontalInverted,n["--n-item-icon-color-active-horizontal"]=r.itemIconColorActiveHorizontalInverted,n["--n-item-icon-color-active-hover-horizontal"]=r.itemIconColorActiveHoverHorizontalInverted,n["--n-item-icon-color-child-active-horizontal"]=r.itemIconColorChildActiveHorizontalInverted,n["--n-item-icon-color-child-active-hover-horizontal"]=r.itemIconColorChildActiveHoverHorizontalInverted,n["--n-arrow-color"]=r.arrowColorInverted,n["--n-arrow-color-hover"]=r.arrowColorHoverInverted,n["--n-arrow-color-active"]=r.arrowColorActiveInverted,n["--n-arrow-color-active-hover"]=r.arrowColorActiveHoverInverted,n["--n-arrow-color-child-active"]=r.arrowColorChildActiveInverted,n["--n-arrow-color-child-active-hover"]=r.arrowColorChildActiveHoverInverted,n["--n-item-color-hover"]=r.itemColorHoverInverted,n["--n-item-color-active"]=r.itemColorActiveInverted,n["--n-item-color-active-hover"]=r.itemColorActiveHoverInverted,n["--n-item-color-active-collapsed"]=r.itemColorActiveCollapsedInverted):(n["--n-group-text-color"]=r.groupTextColor,n["--n-color"]=r.color,n["--n-item-text-color"]=r.itemTextColor,n["--n-item-text-color-hover"]=r.itemTextColorHover,n["--n-item-text-color-active"]=r.itemTextColorActive,n["--n-item-text-color-child-active"]=r.itemTextColorChildActive,n["--n-item-text-color-child-active-hover"]=r.itemTextColorChildActiveHover,n["--n-item-text-color-active-hover"]=r.itemTextColorActiveHover,n["--n-item-icon-color"]=r.itemIconColor,n["--n-item-icon-color-hover"]=r.itemIconColorHover,n["--n-item-icon-color-active"]=r.itemIconColorActive,n["--n-item-icon-color-active-hover"]=r.itemIconColorActiveHover,n["--n-item-icon-color-child-active"]=r.itemIconColorChildActive,n["--n-item-icon-color-child-active-hover"]=r.itemIconColorChildActiveHover,n["--n-item-icon-color-collapsed"]=r.itemIconColorCollapsed,n["--n-item-text-color-horizontal"]=r.itemTextColorHorizontal,n["--n-item-text-color-hover-horizontal"]=r.itemTextColorHoverHorizontal,n["--n-item-text-color-active-horizontal"]=r.itemTextColorActiveHorizontal,n["--n-item-text-color-child-active-horizontal"]=r.itemTextColorChildActiveHorizontal,n["--n-item-text-color-child-active-hover-horizontal"]=r.itemTextColorChildActiveHoverHorizontal,n["--n-item-text-color-active-hover-horizontal"]=r.itemTextColorActiveHoverHorizontal,n["--n-item-icon-color-horizontal"]=r.itemIconColorHorizontal,n["--n-item-icon-color-hover-horizontal"]=r.itemIconColorHoverHorizontal,n["--n-item-icon-color-active-horizontal"]=r.itemIconColorActiveHorizontal,n["--n-item-icon-color-active-hover-horizontal"]=r.itemIconColorActiveHoverHorizontal,n["--n-item-icon-color-child-active-horizontal"]=r.itemIconColorChildActiveHorizontal,n["--n-item-icon-color-child-active-hover-horizontal"]=r.itemIconColorChildActiveHoverHorizontal,n["--n-arrow-color"]=r.arrowColor,n["--n-arrow-color-hover"]=r.arrowColorHover,n["--n-arrow-color-active"]=r.arrowColorActive,n["--n-arrow-color-active-hover"]=r.arrowColorActiveHover,n["--n-arrow-color-child-active"]=r.arrowColorChildActive,n["--n-arrow-color-child-active-hover"]=r.arrowColorChildActiveHover,n["--n-item-color-hover"]=r.itemColorHover,n["--n-item-color-active"]=r.itemColorActive,n["--n-item-color-active-hover"]=r.itemColorActiveHover,n["--n-item-color-active-collapsed"]=r.itemColorActiveCollapsed),n}),y=t?ie("menu",x(()=>e.inverted?"a":"b"),C,e):void 0,D=io(),V=E(null),le=E(null);let H=!0;const ke=()=>{var m;H?H=!1:(m=V.value)===null||m===void 0||m.sync({showAllItemsBeforeCalculate:!0})};function Fe(){return document.getElementById(D)}const oe=E(-1);function Ke(m){oe.value=e.options.length-m}function Ve(m){m||(oe.value=-1)}const De=x(()=>{const m=oe.value;return{children:m===-1?[]:e.options.slice(m)}}),Ue=x(()=>{const{childrenField:m,disabledField:k,keyField:r}=e;return ce([De.value],{getIgnored(g){return he(g)},getChildren(g){return g[m]},getDisabled(g){return g[k]},getKey(g){var $;return($=g[r])!==null&&$!==void 0?$:g.name}})}),Ge=x(()=>ce([{}]).treeNodes[0]);function We(){var m;if(oe.value===-1)return d(ve,{root:!0,level:0,key:"__ellpisisGroupPlaceholder__",internalKey:"__ellpisisGroupPlaceholder__",title:"···",tmNode:Ge.value,domId:D,isEllipsisPlaceholder:!0});const k=Ue.value.treeNodes[0],r=T.value,g=!!(!((m=k.children)===null||m===void 0)&&m.some($=>r.includes($.key)));return d(ve,{level:0,root:!0,key:"__ellpisisGroup__",internalKey:"__ellpisisGroup__",title:"···",virtualChildActive:g,tmNode:k,domId:D,rawNodes:k.rawNode.children||[],tmNodes:k.children||[],isEllipsisPlaceholder:!0})}return{mergedClsPrefix:o,controlledExpandedKeys:P,uncontrolledExpanededKeys:f,mergedExpandedKeys:z,uncontrolledValue:s,mergedValue:j,activePath:T,tmNodes:S,mergedTheme:a,mergedCollapsed:i,cssVars:t?void 0:C,themeClass:y==null?void 0:y.themeClass,overflowRef:V,counterRef:le,updateCounter:()=>{},onResize:ke,onUpdateOverflow:Ve,onUpdateCount:Ke,renderCounter:We,getCounter:Fe,onRender:y==null?void 0:y.onRender,showOption:A,deriveResponsiveState:ke}},render(){const{mergedClsPrefix:e,mode:o,themeClass:t,onRender:a}=this;a==null||a();const l=()=>this.tmNodes.map(c=>we(c,this.$props)),v=o==="horizontal"&&this.responsive,p=()=>d("div",lo(this.$attrs,{role:o==="horizontal"?"menubar":"menu",class:[`${e}-menu`,t,`${e}-menu--${o}`,v&&`${e}-menu--responsive`,this.mergedCollapsed&&`${e}-menu--collapsed`],style:this.cssVars}),v?d(wo,{ref:"overflowRef",onUpdateOverflow:this.onUpdateOverflow,getCounter:this.getCounter,onUpdateCount:this.onUpdateCount,updateCounter:this.updateCounter,style:{width:"100%",display:"flex",overflow:"hidden"}},{default:l,counter:this.renderCounter}):l());return v?d(ro,{onResize:this.onResize},{default:p}):p()}}),Qo={xmlns:"http://www.w3.org/2000/svg","xmlns:xlink":"http://www.w3.org/1999/xlink",viewBox:"0 0 512 512"},Xo=b("path",{d:"M448 256c0-106-86-192-192-192S64 150 64 256s86 192 192 192s192-86 192-192z",fill:"none",stroke:"currentColor","stroke-miterlimit":"10","stroke-width":"32"},null,-1),Zo=b("path",{d:"M250.26 166.05L256 288l5.73-121.95a5.74 5.74 0 0 0-5.79-6h0a5.74 5.74 0 0 0-5.68 6z",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),Jo=b("path",{d:"M256 367.91a20 20 0 1 1 20-20a20 20 0 0 1-20 20z",fill:"currentColor"},null,-1),et=[Xo,Zo,Jo],ot=R({name:"AlertCircleOutline",render:function(o,t){return M(),N("svg",Qo,et)}}),tt={xmlns:"http://www.w3.org/2000/svg","xmlns:xlink":"http://www.w3.org/1999/xlink",viewBox:"0 0 512 512"},rt=b("path",{d:"M32 32v432a16 16 0 0 0 16 16h432",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),nt=b("rect",{x:"96",y:"224",width:"80",height:"192",rx:"20",ry:"20",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),it=b("rect",{x:"240",y:"176",width:"80",height:"240",rx:"20",ry:"20",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),lt=b("rect",{x:"383.64",y:"112",width:"80",height:"304",rx:"20",ry:"20",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),at=[rt,nt,it,lt],Ae=R({name:"BarChartOutline",render:function(o,t){return M(),N("svg",tt,at)}}),st={xmlns:"http://www.w3.org/2000/svg","xmlns:xlink":"http://www.w3.org/1999/xlink",viewBox:"0 0 512 512"},ct=b("path",{d:"M393.87 190a32.1 32.1 0 0 1-45.25 0l-26.57-26.57a32.09 32.09 0 0 1 0-45.26L382.19 58a1 1 0 0 0-.3-1.64c-38.82-16.64-89.15-8.16-121.11 23.57c-30.58 30.35-32.32 76-21.12 115.84a31.93 31.93 0 0 1-9.06 32.08L64 380a48.17 48.17 0 1 0 68 68l153.86-167a31.93 31.93 0 0 1 31.6-9.13c39.54 10.59 84.54 8.6 114.72-21.19c32.49-32 39.5-88.56 23.75-120.93a1 1 0 0 0-1.6-.26z",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-miterlimit":"10","stroke-width":"32"},null,-1),dt=b("circle",{cx:"96",cy:"416",r:"16",fill:"currentColor"},null,-1),ut=[ct,dt],ht=R({name:"BuildOutline",render:function(o,t){return M(),N("svg",st,ut)}}),vt={xmlns:"http://www.w3.org/2000/svg","xmlns:xlink":"http://www.w3.org/1999/xlink",viewBox:"0 0 512 512"},mt=pe('<path d="M80 224l37.78-88.15C123.93 121.5 139.6 112 157.11 112h197.78c17.51 0 33.18 9.5 39.33 23.85L432 224" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M80 224h352v144H80z"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M112 368v32H80v-32"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M432 368v32h-32v-32"></path><circle cx="144" cy="288" r="16" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"></circle><circle cx="368" cy="288" r="16" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"></circle>',6),pt=[mt],ft=R({name:"CarOutline",render:function(o,t){return M(),N("svg",vt,pt)}}),gt={xmlns:"http://www.w3.org/2000/svg","xmlns:xlink":"http://www.w3.org/1999/xlink",viewBox:"0 0 512 512"},xt=b("circle",{cx:"176",cy:"416",r:"16",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),bt=b("circle",{cx:"400",cy:"416",r:"16",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),Ct=b("path",{fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32",d:"M48 80h64l48 272h256"},null,-1),wt=b("path",{d:"M160 288h249.44a8 8 0 0 0 7.85-6.43l28.8-144a8 8 0 0 0-7.85-9.57H128",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),kt=[xt,bt,Ct,wt],yt=R({name:"CartOutline",render:function(o,t){return M(),N("svg",gt,kt)}}),_t={xmlns:"http://www.w3.org/2000/svg","xmlns:xlink":"http://www.w3.org/1999/xlink",viewBox:"0 0 512 512"},zt=b("path",{d:"M436.67 184.11a27.17 27.17 0 0 1-38.3 0l-22.48-22.49a27.15 27.15 0 0 1 0-38.29l50.89-50.89a.85.85 0 0 0-.26-1.38C393.68 57 351.09 64.15 324.05 91c-25.88 25.69-27.35 64.27-17.87 98a27 27 0 0 1-7.67 27.14l-173 160.76a40.76 40.76 0 1 0 57.57 57.54l162.15-173.3a27 27 0 0 1 26.77-7.7c33.46 8.94 71.49 7.26 97.07-17.94c27.49-27.08 33.42-74.94 20.1-102.33a.85.85 0 0 0-1.36-.22z",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-miterlimit":"10","stroke-width":"32"},null,-1),St=b("path",{d:"M224 284c-17.48-17-25.49-24.91-31-30.29a18.24 18.24 0 0 1-3.33-21.35a20.76 20.76 0 0 1 3.5-4.62l15.68-15.29a18.66 18.66 0 0 1 5.63-3.87a18.11 18.11 0 0 1 20 3.62c5.45 5.29 15.43 15 33.41 32.52",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),It=b("path",{d:"M317.07 291.3c40.95 38.1 90.62 83.27 110 99.41a13.46 13.46 0 0 1 .94 19.92L394.63 444a14 14 0 0 1-20.29-.76c-16.53-19.18-61.09-67.11-99.27-107",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),Rt=b("path",{d:"M17.34 193.5l29.41-28.74a4.71 4.71 0 0 1 3.41-1.35a4.85 4.85 0 0 1 3.41 1.35h0a9.86 9.86 0 0 0 8.19 2.77c3.83-.42 7.92-1.6 10.57-4.12c6-5.8-.94-17.23 4.34-24.54a207 207 0 0 1 19.78-22.6c6-5.88 29.84-28.32 69.9-44.45A107.31 107.31 0 0 1 206.67 64c22.59 0 40 10 46.26 15.67a89.54 89.54 0 0 1 10.28 11.64a78.92 78.92 0 0 0-9.21-2.77a68.82 68.82 0 0 0-20-1.26c-13.33 1.09-29.41 7.26-38 14c-13.9 11-19.87 25.72-20.81 44.71c-.68 14.12 2.72 22.1 36.1 55.49a6.6 6.6 0 0 1-.34 9.16l-18.22 18a6.88 6.88 0 0 1-9.54.09c-21.94-21.94-36.65-33.09-45-38.16s-15.07-6.5-18.3-6.85a30.85 30.85 0 0 0-18.27 3.87a11.39 11.39 0 0 0-2.64 2a14.14 14.14 0 0 0 .42 20.08l1.71 1.6a4.63 4.63 0 0 1 0 6.64L71.73 246.6a4.71 4.71 0 0 1-3.41 1.4a4.86 4.86 0 0 1-3.41-1.35l-47.57-46.43a4.88 4.88 0 0 1 0-6.72z",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),At=[zt,St,It,Rt],$t=R({name:"ConstructOutline",render:function(o,t){return M(),N("svg",_t,At)}}),Pt={xmlns:"http://www.w3.org/2000/svg","xmlns:xlink":"http://www.w3.org/1999/xlink",viewBox:"0 0 512 512"},Tt=b("path",{d:"M448 341.37V170.61A32 32 0 0 0 432.11 143l-152-88.46a47.94 47.94 0 0 0-48.24 0L79.89 143A32 32 0 0 0 64 170.61v170.76A32 32 0 0 0 79.89 369l152 88.46a48 48 0 0 0 48.24 0l152-88.46A32 32 0 0 0 448 341.37z",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),jt=b("path",{fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32",d:"M69 153.99l187 110l187-110"},null,-1),Ot=b("path",{fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32",d:"M256 463.99v-200"},null,-1),Mt=[Tt,jt,Ot],Nt=R({name:"CubeOutline",render:function(o,t){return M(),N("svg",Pt,Mt)}}),Bt={xmlns:"http://www.w3.org/2000/svg","xmlns:xlink":"http://www.w3.org/1999/xlink",viewBox:"0 0 512 512"},Ht=b("rect",{x:"48",y:"48",width:"176",height:"176",rx:"20",ry:"20",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),Et=b("rect",{x:"288",y:"48",width:"176",height:"176",rx:"20",ry:"20",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),Lt=b("rect",{x:"48",y:"288",width:"176",height:"176",rx:"20",ry:"20",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),Ft=b("rect",{x:"288",y:"288",width:"176",height:"176",rx:"20",ry:"20",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),Kt=[Ht,Et,Lt,Ft],Vt=R({name:"GridOutline",render:function(o,t){return M(),N("svg",Bt,Kt)}}),Dt={xmlns:"http://www.w3.org/2000/svg","xmlns:xlink":"http://www.w3.org/1999/xlink",viewBox:"0 0 512 512"},Ut=pe('<rect x="80" y="80" width="352" height="352" rx="48" ry="48" fill="none" stroke="currentColor" stroke-linejoin="round" stroke-width="32"></rect><rect x="144" y="144" width="224" height="224" rx="16" ry="16" fill="none" stroke="currentColor" stroke-linejoin="round" stroke-width="32"></rect><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M256 80V48"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M336 80V48"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M176 80V48"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M256 464v-32"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M336 464v-32"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M176 464v-32"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M432 256h32"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M432 336h32"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M432 176h32"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M48 256h32"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M48 336h32"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M48 176h32"></path>',14),Gt=[Ut],Wt=R({name:"HardwareChipOutline",render:function(o,t){return M(),N("svg",Dt,Gt)}}),Yt={xmlns:"http://www.w3.org/2000/svg","xmlns:xlink":"http://www.w3.org/1999/xlink",viewBox:"0 0 512 512"},qt=pe('<path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M368 128h80"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M64 128h240"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M368 384h80"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M64 384h240"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M208 256h240"></path><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M64 256h80"></path><circle cx="336" cy="128" r="32" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"></circle><circle cx="176" cy="256" r="32" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"></circle><circle cx="336" cy="384" r="32" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"></circle>',9),Qt=[qt],Xt=R({name:"OptionsOutline",render:function(o,t){return M(),N("svg",Yt,Qt)}}),Zt={xmlns:"http://www.w3.org/2000/svg","xmlns:xlink":"http://www.w3.org/1999/xlink",viewBox:"0 0 512 512"},Jt=b("path",{d:"M221.09 64a157.09 157.09 0 1 0 157.09 157.09A157.1 157.1 0 0 0 221.09 64z",fill:"none",stroke:"currentColor","stroke-miterlimit":"10","stroke-width":"32"},null,-1),er=b("path",{fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-miterlimit":"10","stroke-width":"32",d:"M338.29 338.29L448 448"},null,-1),or=[Jt,er],tr=R({name:"SearchOutline",render:function(o,t){return M(),N("svg",Zt,or)}}),rr={xmlns:"http://www.w3.org/2000/svg","xmlns:xlink":"http://www.w3.org/1999/xlink",viewBox:"0 0 512 512"},nr=b("path",{d:"M262.29 192.31a64 64 0 1 0 57.4 57.4a64.13 64.13 0 0 0-57.4-57.4zM416.39 256a154.34 154.34 0 0 1-1.53 20.79l45.21 35.46a10.81 10.81 0 0 1 2.45 13.75l-42.77 74a10.81 10.81 0 0 1-13.14 4.59l-44.9-18.08a16.11 16.11 0 0 0-15.17 1.75A164.48 164.48 0 0 1 325 400.8a15.94 15.94 0 0 0-8.82 12.14l-6.73 47.89a11.08 11.08 0 0 1-10.68 9.17h-85.54a11.11 11.11 0 0 1-10.69-8.87l-6.72-47.82a16.07 16.07 0 0 0-9-12.22a155.3 155.3 0 0 1-21.46-12.57a16 16 0 0 0-15.11-1.71l-44.89 18.07a10.81 10.81 0 0 1-13.14-4.58l-42.77-74a10.8 10.8 0 0 1 2.45-13.75l38.21-30a16.05 16.05 0 0 0 6-14.08c-.36-4.17-.58-8.33-.58-12.5s.21-8.27.58-12.35a16 16 0 0 0-6.07-13.94l-38.19-30A10.81 10.81 0 0 1 49.48 186l42.77-74a10.81 10.81 0 0 1 13.14-4.59l44.9 18.08a16.11 16.11 0 0 0 15.17-1.75A164.48 164.48 0 0 1 187 111.2a15.94 15.94 0 0 0 8.82-12.14l6.73-47.89A11.08 11.08 0 0 1 213.23 42h85.54a11.11 11.11 0 0 1 10.69 8.87l6.72 47.82a16.07 16.07 0 0 0 9 12.22a155.3 155.3 0 0 1 21.46 12.57a16 16 0 0 0 15.11 1.71l44.89-18.07a10.81 10.81 0 0 1 13.14 4.58l42.77 74a10.8 10.8 0 0 1-2.45 13.75l-38.21 30a16.05 16.05 0 0 0-6.05 14.08c.33 4.14.55 8.3.55 12.47z",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),ir=[nr],lr=R({name:"SettingsOutline",render:function(o,t){return M(),N("svg",rr,ir)}}),ar={xmlns:"http://www.w3.org/2000/svg","xmlns:xlink":"http://www.w3.org/1999/xlink",viewBox:"0 0 512 512"},sr=b("rect",{x:"64",y:"320",width:"48",height:"160",rx:"8",ry:"8",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),cr=b("rect",{x:"288",y:"224",width:"48",height:"256",rx:"8",ry:"8",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),dr=b("rect",{x:"400",y:"112",width:"48",height:"368",rx:"8",ry:"8",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),ur=b("rect",{x:"176",y:"32",width:"48",height:"448",rx:"8",ry:"8",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),hr=[sr,cr,dr,ur],vr=R({name:"StatsChartOutline",render:function(o,t){return M(),N("svg",ar,hr)}}),mr={xmlns:"http://www.w3.org/2000/svg","xmlns:xlink":"http://www.w3.org/1999/xlink",viewBox:"0 0 512 512"},pr=b("path",{d:"M434.67 285.59v-29.8c0-98.73-80.24-178.79-179.2-178.79a179 179 0 0 0-140.14 67.36m-38.53 82v29.8C76.8 355 157 435 256 435a180.45 180.45 0 0 0 140-66.92",fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32"},null,-1),fr=b("path",{fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32",d:"M32 256l44-44l46 44"},null,-1),gr=b("path",{fill:"none",stroke:"currentColor","stroke-linecap":"round","stroke-linejoin":"round","stroke-width":"32",d:"M480 256l-44 44l-46-44"},null,-1),xr=[pr,fr,gr],br=R({name:"SyncOutline",render:function(o,t){return M(),N("svg",mr,xr)}}),Cr={style:{"font-size":"16px","font-weight":"bold"}},wr=R({__name:"Layout",setup(e){const o=ao(),t=so(),a=mo(),l=E(!1),i=E({}),v=x(()=>t.name),p={AdminDashboard:"管理总览",AdminProduction:"生产管理",AdminQuality:"质量管理",AdminOrders:"订单管理",AdminWarehouse:"仓库管理",AdminMaintenance:"维保管理",AdminAlarms:"异常管理",AdminStatistics:"统计分析",AdminDevices:"设备管理",AdminSettings:"系统设置",AdminERP:"ERP对接",AdminVehicles:"载具管理",AdminLineConfig:"产线配置",AdminSiteConfig:"现场配置"},c=x(()=>p[v.value]||"IGH 管理后台");function s(f){return()=>d(Co,null,{default:()=>d(f)})}const O=x(()=>[{label:"管理总览",key:"AdminDashboard",icon:s(Vt)},{label:"生产管理",key:"AdminProduction",icon:s(Ae),disabled:i.value.Winding===!1},{label:"质量管理",key:"AdminQuality",icon:s(Ae),disabled:i.value.Quality===!1},{label:"订单管理",key:"AdminOrders",icon:s(yt),disabled:i.value.Packaging===!1},{label:"仓库管理",key:"AdminWarehouse",icon:s(Nt),disabled:i.value.Warehouse===!1},{label:"维保管理",key:"AdminMaintenance",icon:s(ht),disabled:i.value.Maintenance===!1},{label:"ERP对接",key:"AdminERP",icon:s(br),disabled:i.value.ERP===!1},{label:"载具管理",key:"AdminVehicles",icon:s(ft),disabled:i.value.Vehicle===!1},{label:"设备管理",key:"AdminDevices",icon:s(Wt)},{label:"产线配置",key:"AdminLineConfig",icon:s(Xt)},{label:"现场配置",key:"AdminSiteConfig",icon:s($t)},{label:"异常管理",key:"AdminAlarms",icon:s(ot)},{label:"全链追溯",key:"AdminTrace",icon:s(tr)},{label:"统计分析",key:"AdminStatistics",icon:s(vr)},{label:"系统设置",key:"AdminSettings",icon:s(lr)}]);function j(){a.logout(),o.push("/login")}return co(async()=>{try{i.value=(await fo.get("/admin/modules")).data||{}}catch{}}),(f,w)=>{const P=qo,z=Eo,S=yo,T=_o,Y=Oo,F=uo("router-view"),B=Po,A=$o;return M(),ho(A,{"has-sider":"",style:{height:"100vh"}},{default:U(()=>[K(z,{bordered:"","collapse-mode":"width","collapsed-width":64,width:200,collapsed:l.value,"show-trigger":"",onCollapse:w[1]||(w[1]=C=>l.value=!0),onExpand:w[2]||(w[2]=C=>l.value=!1)},{default:U(()=>[w[4]||(w[4]=b("div",{class:"logo"},"IGH",-1)),K(P,{collapsed:l.value,options:O.value,value:v.value,"onUpdate:value":w[0]||(w[0]=C=>ze(o).push({name:C}))},null,8,["collapsed","options","value"])]),_:1},8,["collapsed"]),K(A,null,{default:U(()=>[K(Y,{bordered:"",style:{height:"52px",display:"flex","align-items":"center",padding:"0 20px","justify-content":"space-between"}},{default:U(()=>[b("span",Cr,vo(c.value),1),K(T,null,{default:U(()=>[K(S,{text:"",size:"small",onClick:w[3]||(w[3]=C=>ze(o).push("/board/overview"))},{default:U(()=>[...w[5]||(w[5]=[Se("看板",-1)])]),_:1}),K(S,{text:"",size:"small",onClick:j},{default:U(()=>[...w[6]||(w[6]=[Se("退出",-1)])]),_:1})]),_:1})]),_:1}),K(B,{style:{padding:"16px",overflow:"auto",height:"calc(100vh - 52px)"}},{default:U(()=>[K(po),K(F)]),_:1})]),_:1})]),_:1})}}}),Kr=zo(wr,[["__scopeId","data-v-a6200780"]]);export{Kr as default};
