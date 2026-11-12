import{N as ve,p as de,B as Ie,b as Ke,V as _e,h as pe,r as $e,e as ze}from"./Popover-KSUJUYtz.js";import{b4 as Te,b5 as je,b0 as q,bj as De,b2 as G,bX as se,bi as Fe,bk as T,a9 as N,au as a,bI as ae,bS as H,bF as Ae,Q as m,a1 as le,bl as J,ay as B,u as P,x as k,t as z,bW as be,aW as ce,av as Be,bT as me,k as Me,bJ as X,bf as E,F as Le,X as Ee,y as fe,w as $,aS as He,ah as Ue,bC as I,a2 as A}from"./index--k3XO6sF.js";import{p as we,m as We,d as Ve,f as qe}from"./fade-in-scale-up.cssr-DdQzqyjR.js";import{f as Ge,u as Xe}from"./get-D7KjFPV7.js";import{h as Je}from"./utils-DGCgOF5A.js";import{c as re}from"./resolve-slot-BxqMXCcO.js";function Qe(e={},n){const i=Fe({ctrl:!1,command:!1,win:!1,shift:!1,tab:!1}),{keydown:t,keyup:r}=e,o=s=>{switch(s.key){case"Control":i.ctrl=!0;break;case"Meta":i.command=!0,i.win=!0;break;case"Shift":i.shift=!0;break;case"Tab":i.tab=!0;break}t!==void 0&&Object.keys(t).forEach(w=>{if(w!==s.key)return;const v=t[w];if(typeof v=="function")v(s);else{const{stop:g=!1,prevent:x=!1}=v;g&&s.stopPropagation(),x&&s.preventDefault(),v.handler(s)}})},l=s=>{switch(s.key){case"Control":i.ctrl=!1;break;case"Meta":i.command=!1,i.win=!1;break;case"Shift":i.shift=!1;break;case"Tab":i.tab=!1;break}r!==void 0&&Object.keys(r).forEach(w=>{if(w!==s.key)return;const v=r[w];if(typeof v=="function")v(s);else{const{stop:g=!1,prevent:x=!1}=v;g&&s.stopPropagation(),x&&s.preventDefault(),v.handler(s)}})},u=()=>{(n===void 0||n.value)&&(G("keydown",document,o),G("keyup",document,l)),n!==void 0&&se(n,s=>{s?(G("keydown",document,o),G("keyup",document,l)):(q("keydown",document,o),q("keyup",document,l))})};return Je()?(Te(u),je(()=>{(n===void 0||n.value)&&(q("keydown",document,o),q("keyup",document,l))})):u(),De(i)}function Ze(e,n,i){const t=T(e.value);let r=null;return se(e,o=>{r!==null&&window.clearTimeout(r),o===!0?i&&!i.value?t.value=!0:r=window.setTimeout(()=>{t.value=!0},n):t.value=!1}),t}function Ye(e){return n=>{n?e.value=n.$el:e.value=null}}const eo=N({name:"ChevronRight",render(){return a("svg",{viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg"},a("path",{d:"M5.64645 3.14645C5.45118 3.34171 5.45118 3.65829 5.64645 3.85355L9.79289 8L5.64645 12.1464C5.45118 12.3417 5.45118 12.6583 5.64645 12.8536C5.84171 13.0488 6.15829 13.0488 6.35355 12.8536L10.8536 8.35355C11.0488 8.15829 11.0488 7.84171 10.8536 7.64645L6.35355 3.14645C6.15829 2.95118 5.84171 2.95118 5.64645 3.14645Z",fill:"currentColor"}))}}),oo=Object.assign(Object.assign({},de),H.props),xo=N({name:"Tooltip",props:oo,slots:Object,__popover__:!0,setup(e){const{mergedClsPrefixRef:n}=ae(e),i=H("Tooltip","-tooltip",void 0,Ae,e,n),t=T(null);return Object.assign(Object.assign({},{syncPosition(){t.value.syncPosition()},setShow(o){t.value.setShow(o)}}),{popoverRef:t,mergedTheme:i,popoverThemeOverrides:m(()=>i.value.self)})},render(){const{mergedTheme:e,internalExtraClass:n}=this;return a(ve,Object.assign(Object.assign({},this.$props),{theme:e.peers.Popover,themeOverrides:e.peerOverrides.Popover,builtinThemeOverrides:this.popoverThemeOverrides,internalExtraClass:n.concat("tooltip"),ref:"popoverRef"}),this.$slots)}}),ue=le("n-dropdown-menu"),Q=le("n-dropdown"),he=le("n-dropdown-option"),ye=N({name:"DropdownDivider",props:{clsPrefix:{type:String,required:!0}},render(){return a("div",{class:`${this.clsPrefix}-dropdown-divider`})}}),no=N({name:"DropdownGroupHeader",props:{clsPrefix:{type:String,required:!0},tmNode:{type:Object,required:!0}},setup(){const{showIconRef:e,hasSubmenuRef:n}=B(ue),{renderLabelRef:i,labelFieldRef:t,nodePropsRef:r,renderOptionRef:o}=B(Q);return{labelField:t,showIcon:e,hasSubmenu:n,renderLabel:i,nodeProps:r,renderOption:o}},render(){var e;const{clsPrefix:n,hasSubmenu:i,showIcon:t,nodeProps:r,renderLabel:o,renderOption:l}=this,{rawNode:u}=this.tmNode,s=a("div",Object.assign({class:`${n}-dropdown-option`},r==null?void 0:r(u)),a("div",{class:`${n}-dropdown-option-body ${n}-dropdown-option-body--group`},a("div",{"data-dropdown-option":!0,class:[`${n}-dropdown-option-body__prefix`,t&&`${n}-dropdown-option-body__prefix--show-icon`]},J(u.icon)),a("div",{class:`${n}-dropdown-option-body__label`,"data-dropdown-option":!0},o?o(u):J((e=u.title)!==null&&e!==void 0?e:u[this.labelField])),a("div",{class:[`${n}-dropdown-option-body__suffix`,i&&`${n}-dropdown-option-body__suffix--has-submenu`],"data-dropdown-option":!0})));return l?l({node:s,option:u}):s}}),to=P("icon",`
 height: 1em;
 width: 1em;
 line-height: 1em;
 text-align: center;
 display: inline-block;
 position: relative;
 fill: currentColor;
`,[k("color-transition",{transition:"color .3s var(--n-bezier)"}),k("depth",{color:"var(--n-color)"},[z("svg",{opacity:"var(--n-opacity)",transition:"opacity .3s var(--n-bezier)"})]),z("svg",{height:"1em",width:"1em"})]),ro=Object.assign(Object.assign({},H.props),{depth:[String,Number],size:[Number,String],color:String,component:[Object,Function]}),io=N({_n_icon__:!0,name:"Icon",inheritAttrs:!1,props:ro,setup(e){const{mergedClsPrefixRef:n,inlineThemeDisabled:i}=ae(e),t=H("Icon","-icon",to,Be,e,n),r=m(()=>{const{depth:l}=e,{common:{cubicBezierEaseInOut:u},self:s}=t.value;if(l!==void 0){const{color:w,[`opacity${l}Depth`]:v}=s;return{"--n-bezier":u,"--n-color":w,"--n-opacity":v}}return{"--n-bezier":u,"--n-color":"","--n-opacity":""}}),o=i?me("icon",m(()=>`${e.depth||"d"}`),r,e):void 0;return{mergedClsPrefix:n,mergedStyle:m(()=>{const{size:l,color:u}=e;return{fontSize:Ge(l),color:u}}),cssVars:i?void 0:r,themeClass:o==null?void 0:o.themeClass,onRender:o==null?void 0:o.onRender}},render(){var e;const{$parent:n,depth:i,mergedClsPrefix:t,component:r,onRender:o,themeClass:l}=this;return!((e=n==null?void 0:n.$options)===null||e===void 0)&&e._n_icon__&&be("icon","don't wrap `n-icon` inside `n-icon`"),o==null||o(),a("i",ce(this.$attrs,{role:"img",class:[`${t}-icon`,l,{[`${t}-icon--depth`]:i,[`${t}-icon--color-transition`]:i!==void 0}],style:[this.cssVars,this.mergedStyle]}),r?a(r):this.$slots)}});function ie(e,n){return e.type==="submenu"||e.type===void 0&&e[n]!==void 0}function so(e){return e.type==="group"}function ge(e){return e.type==="divider"}function ao(e){return e.type==="render"}const xe=N({name:"DropdownOption",props:{clsPrefix:{type:String,required:!0},tmNode:{type:Object,required:!0},parentKey:{type:[String,Number],default:null},placement:{type:String,default:"right-start"},props:Object,scrollable:Boolean},setup(e){const n=B(Q),{hoverKeyRef:i,keyboardKeyRef:t,lastToggledSubmenuKeyRef:r,pendingKeyPathRef:o,activeKeyPathRef:l,animatedRef:u,mergedShowRef:s,renderLabelRef:w,renderIconRef:v,labelFieldRef:g,childrenFieldRef:x,renderOptionRef:R,nodePropsRef:C,menuPropsRef:j}=n,S=B(he,null),K=B(ue),_=B(we),W=m(()=>e.tmNode.rawNode),U=m(()=>{const{value:d}=x;return ie(e.tmNode.rawNode,d)}),Z=m(()=>{const{disabled:d}=e.tmNode;return d}),Y=m(()=>{if(!U.value)return!1;const{key:d,disabled:f}=e.tmNode;if(f)return!1;const{value:y}=i,{value:D}=t,{value:te}=r,{value:F}=o;return y!==null?F.includes(d):D!==null?F.includes(d)&&F[F.length-1]!==d:te!==null?F.includes(d):!1}),ee=m(()=>t.value===null&&!u.value),oe=Ze(Y,300,ee),ne=m(()=>!!(S!=null&&S.enteringSubmenuRef.value)),M=T(!1);E(he,{enteringSubmenuRef:M});function L(){M.value=!0}function V(){M.value=!1}function O(){const{parentKey:d,tmNode:f}=e;f.disabled||s.value&&(r.value=d,t.value=null,i.value=f.key)}function c(){const{tmNode:d}=e;d.disabled||s.value&&i.value!==d.key&&O()}function p(d){if(e.tmNode.disabled||!s.value)return;const{relatedTarget:f}=d;f&&!pe({target:f},"dropdownOption")&&!pe({target:f},"scrollbarRail")&&(i.value=null)}function h(){const{value:d}=U,{tmNode:f}=e;s.value&&!d&&!f.disabled&&(n.doSelect(f.key,f.rawNode),n.doUpdateShow(!1))}return{labelField:g,renderLabel:w,renderIcon:v,siblingHasIcon:K.showIconRef,siblingHasSubmenu:K.hasSubmenuRef,menuProps:j,popoverBody:_,animated:u,mergedShowSubmenu:m(()=>oe.value&&!ne.value),rawNode:W,hasSubmenu:U,pending:X(()=>{const{value:d}=o,{key:f}=e.tmNode;return d.includes(f)}),childActive:X(()=>{const{value:d}=l,{key:f}=e.tmNode,y=d.findIndex(D=>f===D);return y===-1?!1:y<d.length-1}),active:X(()=>{const{value:d}=l,{key:f}=e.tmNode,y=d.findIndex(D=>f===D);return y===-1?!1:y===d.length-1}),mergedDisabled:Z,renderOption:R,nodeProps:C,handleClick:h,handleMouseMove:c,handleMouseEnter:O,handleMouseLeave:p,handleSubmenuBeforeEnter:L,handleSubmenuAfterEnter:V}},render(){var e,n;const{animated:i,rawNode:t,mergedShowSubmenu:r,clsPrefix:o,siblingHasIcon:l,siblingHasSubmenu:u,renderLabel:s,renderIcon:w,renderOption:v,nodeProps:g,props:x,scrollable:R}=this;let C=null;if(r){const _=(e=this.menuProps)===null||e===void 0?void 0:e.call(this,t,t.children);C=a(Se,Object.assign({},_,{clsPrefix:o,scrollable:this.scrollable,tmNodes:this.tmNode.children,parentKey:this.tmNode.key}))}const j={class:[`${o}-dropdown-option-body`,this.pending&&`${o}-dropdown-option-body--pending`,this.active&&`${o}-dropdown-option-body--active`,this.childActive&&`${o}-dropdown-option-body--child-active`,this.mergedDisabled&&`${o}-dropdown-option-body--disabled`],onMousemove:this.handleMouseMove,onMouseenter:this.handleMouseEnter,onMouseleave:this.handleMouseLeave,onClick:this.handleClick},S=g==null?void 0:g(t),K=a("div",Object.assign({class:[`${o}-dropdown-option`,S==null?void 0:S.class],"data-dropdown-option":!0},S),a("div",ce(j,x),[a("div",{class:[`${o}-dropdown-option-body__prefix`,l&&`${o}-dropdown-option-body__prefix--show-icon`]},[w?w(t):J(t.icon)]),a("div",{"data-dropdown-option":!0,class:`${o}-dropdown-option-body__label`},s?s(t):J((n=t[this.labelField])!==null&&n!==void 0?n:t.title)),a("div",{"data-dropdown-option":!0,class:[`${o}-dropdown-option-body__suffix`,u&&`${o}-dropdown-option-body__suffix--has-submenu`]},this.hasSubmenu?a(io,null,{default:()=>a(eo,null)}):null)]),this.hasSubmenu?a(Ie,null,{default:()=>[a(Ke,null,{default:()=>a("div",{class:`${o}-dropdown-offset-container`},a(_e,{show:this.mergedShowSubmenu,placement:this.placement,to:R&&this.popoverBody||void 0,teleportDisabled:!R},{default:()=>a("div",{class:`${o}-dropdown-menu-wrapper`},i?a(Me,{onBeforeEnter:this.handleSubmenuBeforeEnter,onAfterEnter:this.handleSubmenuAfterEnter,name:"fade-in-scale-up-transition",appear:!0},{default:()=>C}):C)}))})]}):null);return v?v({node:K,option:t}):K}}),lo=N({name:"NDropdownGroup",props:{clsPrefix:{type:String,required:!0},tmNode:{type:Object,required:!0},parentKey:{type:[String,Number],default:null}},render(){const{tmNode:e,parentKey:n,clsPrefix:i}=this,{children:t}=e;return a(Le,null,a(no,{clsPrefix:i,tmNode:e,key:e.key}),t==null?void 0:t.map(r=>{const{rawNode:o}=r;return o.show===!1?null:ge(o)?a(ye,{clsPrefix:i,key:r.key}):r.isGroup?(be("dropdown","`group` node is not allowed to be put in `group` node."),null):a(xe,{clsPrefix:i,tmNode:r,parentKey:n,key:r.key})}))}}),co=N({name:"DropdownRenderOption",props:{tmNode:{type:Object,required:!0}},render(){const{rawNode:{render:e,props:n}}=this.tmNode;return a("div",n,[e==null?void 0:e()])}}),Se=N({name:"DropdownMenu",props:{scrollable:Boolean,showArrow:Boolean,arrowStyle:[String,Object],clsPrefix:{type:String,required:!0},tmNodes:{type:Array,default:()=>[]},parentKey:{type:[String,Number],default:null}},setup(e){const{renderIconRef:n,childrenFieldRef:i}=B(Q);E(ue,{showIconRef:m(()=>{const r=n.value;return e.tmNodes.some(o=>{var l;if(o.isGroup)return(l=o.children)===null||l===void 0?void 0:l.some(({rawNode:s})=>r?r(s):s.icon);const{rawNode:u}=o;return r?r(u):u.icon})}),hasSubmenuRef:m(()=>{const{value:r}=i;return e.tmNodes.some(o=>{var l;if(o.isGroup)return(l=o.children)===null||l===void 0?void 0:l.some(({rawNode:s})=>ie(s,r));const{rawNode:u}=o;return ie(u,r)})})});const t=T(null);return E(We,null),E(Ve,null),E(we,t),{bodyRef:t}},render(){const{parentKey:e,clsPrefix:n,scrollable:i}=this,t=this.tmNodes.map(r=>{const{rawNode:o}=r;return o.show===!1?null:ao(o)?a(co,{tmNode:r,key:r.key}):ge(o)?a(ye,{clsPrefix:n,key:r.key}):so(o)?a(lo,{clsPrefix:n,tmNode:r,parentKey:e,key:r.key}):a(xe,{clsPrefix:n,tmNode:r,parentKey:e,key:r.key,props:o.props,scrollable:i})});return a("div",{class:[`${n}-dropdown-menu`,i&&`${n}-dropdown-menu--scrollable`],ref:"bodyRef"},i?a(Ee,{contentClass:`${n}-dropdown-menu__content`},{default:()=>t}):t,this.showArrow?$e({clsPrefix:n,arrowStyle:this.arrowStyle,arrowClass:void 0,arrowWrapperClass:void 0,arrowWrapperStyle:void 0}):null)}}),uo=P("dropdown-menu",`
 transform-origin: var(--v-transform-origin);
 background-color: var(--n-color);
 border-radius: var(--n-border-radius);
 box-shadow: var(--n-box-shadow);
 position: relative;
 transition:
 background-color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier);
`,[qe(),P("dropdown-option",`
 position: relative;
 `,[z("a",`
 text-decoration: none;
 color: inherit;
 outline: none;
 `,[z("&::before",`
 content: "";
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 `)]),P("dropdown-option-body",`
 display: flex;
 cursor: pointer;
 position: relative;
 height: var(--n-option-height);
 line-height: var(--n-option-height);
 font-size: var(--n-font-size);
 color: var(--n-option-text-color);
 transition: color .3s var(--n-bezier);
 `,[z("&::before",`
 content: "";
 position: absolute;
 top: 0;
 bottom: 0;
 left: 4px;
 right: 4px;
 transition: background-color .3s var(--n-bezier);
 border-radius: var(--n-border-radius);
 `),fe("disabled",[k("pending",`
 color: var(--n-option-text-color-hover);
 `,[$("prefix, suffix",`
 color: var(--n-option-text-color-hover);
 `),z("&::before","background-color: var(--n-option-color-hover);")]),k("active",`
 color: var(--n-option-text-color-active);
 `,[$("prefix, suffix",`
 color: var(--n-option-text-color-active);
 `),z("&::before","background-color: var(--n-option-color-active);")]),k("child-active",`
 color: var(--n-option-text-color-child-active);
 `,[$("prefix, suffix",`
 color: var(--n-option-text-color-child-active);
 `)])]),k("disabled",`
 cursor: not-allowed;
 opacity: var(--n-option-opacity-disabled);
 `),k("group",`
 font-size: calc(var(--n-font-size) - 1px);
 color: var(--n-group-header-text-color);
 `,[$("prefix",`
 width: calc(var(--n-option-prefix-width) / 2);
 `,[k("show-icon",`
 width: calc(var(--n-option-icon-prefix-width) / 2);
 `)])]),$("prefix",`
 width: var(--n-option-prefix-width);
 display: flex;
 justify-content: center;
 align-items: center;
 color: var(--n-prefix-color);
 transition: color .3s var(--n-bezier);
 z-index: 1;
 `,[k("show-icon",`
 width: var(--n-option-icon-prefix-width);
 `),P("icon",`
 font-size: var(--n-option-icon-size);
 `)]),$("label",`
 white-space: nowrap;
 flex: 1;
 z-index: 1;
 `),$("suffix",`
 box-sizing: border-box;
 flex-grow: 0;
 flex-shrink: 0;
 display: flex;
 justify-content: flex-end;
 align-items: center;
 min-width: var(--n-option-suffix-width);
 padding: 0 8px;
 transition: color .3s var(--n-bezier);
 color: var(--n-suffix-color);
 z-index: 1;
 `,[k("has-submenu",`
 width: var(--n-option-icon-suffix-width);
 `),P("icon",`
 font-size: var(--n-option-icon-size);
 `)]),P("dropdown-menu","pointer-events: all;")]),P("dropdown-offset-container",`
 pointer-events: none;
 position: absolute;
 left: 0;
 right: 0;
 top: -4px;
 bottom: -4px;
 `)]),P("dropdown-divider",`
 transition: background-color .3s var(--n-bezier);
 background-color: var(--n-divider-color);
 height: 1px;
 margin: 4px 0;
 `),P("dropdown-menu-wrapper",`
 transform-origin: var(--v-transform-origin);
 width: fit-content;
 `),z(">",[P("scrollbar",`
 height: inherit;
 max-height: inherit;
 `)]),fe("scrollable",`
 padding: var(--n-padding);
 `),k("scrollable",[$("content",`
 padding: var(--n-padding);
 `)])]),po={animated:{type:Boolean,default:!0},keyboard:{type:Boolean,default:!0},size:String,inverted:Boolean,placement:{type:String,default:"bottom"},onSelect:[Function,Array],options:{type:Array,default:()=>[]},menuProps:Function,showArrow:Boolean,renderLabel:Function,renderIcon:Function,renderOption:Function,nodeProps:Function,labelField:{type:String,default:"label"},keyField:{type:String,default:"key"},childrenField:{type:String,default:"children"},value:[String,Number]},fo=Object.keys(de),ho=Object.assign(Object.assign(Object.assign({},de),po),H.props),So=N({name:"Dropdown",inheritAttrs:!1,props:ho,setup(e){const n=T(!1),i=Xe(I(e,"show"),n),t=m(()=>{const{keyField:c,childrenField:p}=e;return ze(e.options,{getKey(h){return h[c]},getDisabled(h){return h.disabled===!0},getIgnored(h){return h.type==="divider"||h.type==="render"},getChildren(h){return h[p]}})}),r=m(()=>t.value.treeNodes),o=T(null),l=T(null),u=T(null),s=m(()=>{var c,p,h;return(h=(p=(c=o.value)!==null&&c!==void 0?c:l.value)!==null&&p!==void 0?p:u.value)!==null&&h!==void 0?h:null}),w=m(()=>t.value.getPath(s.value).keyPath),v=m(()=>t.value.getPath(e.value).keyPath),g=X(()=>e.keyboard&&i.value);Qe({keydown:{ArrowUp:{prevent:!0,handler:ee},ArrowRight:{prevent:!0,handler:Y},ArrowDown:{prevent:!0,handler:oe},ArrowLeft:{prevent:!0,handler:Z},Enter:{prevent:!0,handler:ne},Escape:U}},g);const{mergedClsPrefixRef:x,inlineThemeDisabled:R,mergedComponentPropsRef:C}=ae(e),j=m(()=>{var c,p;return e.size||((p=(c=C==null?void 0:C.value)===null||c===void 0?void 0:c.Dropdown)===null||p===void 0?void 0:p.size)||"medium"}),S=H("Dropdown","-dropdown",uo,Ue,e,x);E(Q,{labelFieldRef:I(e,"labelField"),childrenFieldRef:I(e,"childrenField"),renderLabelRef:I(e,"renderLabel"),renderIconRef:I(e,"renderIcon"),hoverKeyRef:o,keyboardKeyRef:l,lastToggledSubmenuKeyRef:u,pendingKeyPathRef:w,activeKeyPathRef:v,animatedRef:I(e,"animated"),mergedShowRef:i,nodePropsRef:I(e,"nodeProps"),renderOptionRef:I(e,"renderOption"),menuPropsRef:I(e,"menuProps"),doSelect:K,doUpdateShow:_}),se(i,c=>{!e.animated&&!c&&W()});function K(c,p){const{onSelect:h}=e;h&&re(h,c,p)}function _(c){const{"onUpdate:show":p,onUpdateShow:h}=e;p&&re(p,c),h&&re(h,c),n.value=c}function W(){o.value=null,l.value=null,u.value=null}function U(){_(!1)}function Z(){L("left")}function Y(){L("right")}function ee(){L("up")}function oe(){L("down")}function ne(){const c=M();c!=null&&c.isLeaf&&i.value&&(K(c.key,c.rawNode),_(!1))}function M(){var c;const{value:p}=t,{value:h}=s;return!p||h===null?null:(c=p.getNode(h))!==null&&c!==void 0?c:null}function L(c){const{value:p}=s,{value:{getFirstAvailableNode:h}}=t;let d=null;if(p===null){const f=h();f!==null&&(d=f.key)}else{const f=M();if(f){let y;switch(c){case"down":y=f.getNext();break;case"up":y=f.getPrev();break;case"right":y=f.getChild();break;case"left":y=f.getParent();break}y&&(d=y.key)}}d!==null&&(o.value=null,l.value=d)}const V=m(()=>{const{inverted:c}=e,p=j.value,{common:{cubicBezierEaseInOut:h},self:d}=S.value,{padding:f,dividerColor:y,borderRadius:D,optionOpacityDisabled:te,[A("optionIconSuffixWidth",p)]:F,[A("optionSuffixWidth",p)]:Pe,[A("optionIconPrefixWidth",p)]:ke,[A("optionPrefixWidth",p)]:Ne,[A("fontSize",p)]:Re,[A("optionHeight",p)]:Ce,[A("optionIconSize",p)]:Oe}=d,b={"--n-bezier":h,"--n-font-size":Re,"--n-padding":f,"--n-border-radius":D,"--n-option-height":Ce,"--n-option-prefix-width":Ne,"--n-option-icon-prefix-width":ke,"--n-option-suffix-width":Pe,"--n-option-icon-suffix-width":F,"--n-option-icon-size":Oe,"--n-divider-color":y,"--n-option-opacity-disabled":te};return c?(b["--n-color"]=d.colorInverted,b["--n-option-color-hover"]=d.optionColorHoverInverted,b["--n-option-color-active"]=d.optionColorActiveInverted,b["--n-option-text-color"]=d.optionTextColorInverted,b["--n-option-text-color-hover"]=d.optionTextColorHoverInverted,b["--n-option-text-color-active"]=d.optionTextColorActiveInverted,b["--n-option-text-color-child-active"]=d.optionTextColorChildActiveInverted,b["--n-prefix-color"]=d.prefixColorInverted,b["--n-suffix-color"]=d.suffixColorInverted,b["--n-group-header-text-color"]=d.groupHeaderTextColorInverted):(b["--n-color"]=d.color,b["--n-option-color-hover"]=d.optionColorHover,b["--n-option-color-active"]=d.optionColorActive,b["--n-option-text-color"]=d.optionTextColor,b["--n-option-text-color-hover"]=d.optionTextColorHover,b["--n-option-text-color-active"]=d.optionTextColorActive,b["--n-option-text-color-child-active"]=d.optionTextColorChildActive,b["--n-prefix-color"]=d.prefixColor,b["--n-suffix-color"]=d.suffixColor,b["--n-group-header-text-color"]=d.groupHeaderTextColor),b}),O=R?me("dropdown",m(()=>`${j.value[0]}${e.inverted?"i":""}`),V,e):void 0;return{mergedClsPrefix:x,mergedTheme:S,mergedSize:j,tmNodes:r,mergedShow:i,handleAfterLeave:()=>{e.animated&&W()},doUpdateShow:_,cssVars:R?void 0:V,themeClass:O==null?void 0:O.themeClass,onRender:O==null?void 0:O.onRender}},render(){const e=(t,r,o,l,u)=>{var s;const{mergedClsPrefix:w,menuProps:v}=this;(s=this.onRender)===null||s===void 0||s.call(this);const g=(v==null?void 0:v(void 0,this.tmNodes.map(R=>R.rawNode)))||{},x={ref:Ye(r),class:[t,`${w}-dropdown`,`${w}-dropdown--${this.mergedSize}-size`,this.themeClass],clsPrefix:w,tmNodes:this.tmNodes,style:[...o,this.cssVars],showArrow:this.showArrow,arrowStyle:this.arrowStyle,scrollable:this.scrollable,onMouseenter:l,onMouseleave:u};return a(Se,ce(this.$attrs,x,g))},{mergedTheme:n}=this,i={show:this.mergedShow,theme:n.peers.Popover,themeOverrides:n.peerOverrides.Popover,internalOnAfterLeave:this.handleAfterLeave,internalRenderBody:e,onUpdateShow:this.doUpdateShow,"onUpdate:show":void 0};return a(ve,Object.assign({},He(this.$props,fo),i),{trigger:()=>{var t,r;return(r=(t=this.$slots).default)===null||r===void 0?void 0:r.call(t)}})}});export{eo as C,So as N,io as a,xo as b,Ye as c};
