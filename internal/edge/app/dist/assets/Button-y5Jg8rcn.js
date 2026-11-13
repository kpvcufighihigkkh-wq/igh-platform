import{b5 as ne,ay as ie,Q as T,a1 as ae,bf as ke,t as f,H as Re,u as N,a9 as se,au as p,bR as Ie,aZ as Fe,bC as He,bk as H,P as le,x,w as u,y as ee,aw as Ee,e as Oe,bJ as je,bI as De,bS as de,bP as _e,bT as Ne,f as Ke,d as Me,s as Ve,a2 as r,A as D}from"./index-Z9hz6zvP.js";import{i as K}from"./is-browser-DqcmxZSF.js";import{b as oe,c as Ge,i as We}from"./resolve-slot-C44M-6Fz.js";import{c as re}from"./color-to-class-B0iQgAn2.js";const te=ae("n-form-item");function Qe(e,{defaultSize:h="medium",mergedSize:v,mergedDisabled:d}={}){const n=ie(te,null);ke(te,null);const M=T(v?()=>v(n):()=>{const{size:m}=e;if(m)return m;if(n){const{mergedSize:P}=n;if(P.value!==void 0)return P.value}return h}),E=T(d?()=>d(n):()=>{const{disabled:m}=e;return m!==void 0?m:n?n.disabled.value:!1}),F=T(()=>{const{status:m}=e;return m||(n==null?void 0:n.mergedValidationStatus.value)});return ne(()=>{n&&n.restoreValidation()}),{mergedSizeRef:M,mergedDisabledRef:E,mergedStatusRef:F,nTriggerFormBlur(){n&&n.handleContentBlur()},nTriggerFormChange(){n&&n.handleContentChange()},nTriggerFormFocus(){n&&n.handleContentFocus()},nTriggerFormInput(){n&&n.handleContentInput()}}}const{cubicBezierEaseInOut:w}=Re;function qe({duration:e=".2s",delay:h=".1s"}={}){return[f("&.fade-in-width-expand-transition-leave-from, &.fade-in-width-expand-transition-enter-to",{opacity:1}),f("&.fade-in-width-expand-transition-leave-to, &.fade-in-width-expand-transition-enter-from",`
 opacity: 0!important;
 margin-left: 0!important;
 margin-right: 0!important;
 `),f("&.fade-in-width-expand-transition-leave-active",`
 overflow: hidden;
 transition:
 opacity ${e} ${w},
 max-width ${e} ${w} ${h},
 margin-left ${e} ${w} ${h},
 margin-right ${e} ${w} ${h};
 `),f("&.fade-in-width-expand-transition-enter-active",`
 overflow: hidden;
 transition:
 opacity ${e} ${w} ${h},
 max-width ${e} ${w},
 margin-left ${e} ${w},
 margin-right ${e} ${w};
 `)]}const Ae=N("base-wave",`
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 border-radius: inherit;
`),Le=se({name:"BaseWave",props:{clsPrefix:{type:String,required:!0}},setup(e){Ie("-base-wave",Ae,He(e,"clsPrefix"));const h=H(null),v=H(!1);let d=null;return ne(()=>{d!==null&&window.clearTimeout(d)}),{active:v,selfRef:h,play(){d!==null&&(window.clearTimeout(d),v.value=!1,d=null),Fe(()=>{var n;(n=h.value)===null||n===void 0||n.offsetHeight,v.value=!0,d=window.setTimeout(()=>{v.value=!1,d=null},1e3)})}}},render(){const{clsPrefix:e}=this;return p("div",{ref:"selfRef","aria-hidden":!0,class:[`${e}-base-wave`,this.active&&`${e}-base-wave--active`]})}}),Xe=K&&"chrome"in window;K&&navigator.userAgent.includes("Firefox");const Ye=K&&navigator.userAgent.includes("Safari")&&!Xe;function B(e){return le(e,[255,255,255,.16])}function _(e){return le(e,[0,0,0,.12])}const Je=ae("n-button-group"),Ue=f([N("button",`
 margin: 0;
 font-weight: var(--n-font-weight);
 line-height: 1;
 font-family: inherit;
 padding: var(--n-padding);
 height: var(--n-height);
 font-size: var(--n-font-size);
 border-radius: var(--n-border-radius);
 color: var(--n-text-color);
 background-color: var(--n-color);
 width: var(--n-width);
 white-space: nowrap;
 outline: none;
 position: relative;
 z-index: auto;
 border: none;
 display: inline-flex;
 flex-wrap: nowrap;
 flex-shrink: 0;
 align-items: center;
 justify-content: center;
 user-select: none;
 -webkit-user-select: none;
 text-align: center;
 cursor: pointer;
 text-decoration: none;
 transition:
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 opacity .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 `,[x("color",[u("border",{borderColor:"var(--n-border-color)"}),x("disabled",[u("border",{borderColor:"var(--n-border-color-disabled)"})]),ee("disabled",[f("&:focus",[u("state-border",{borderColor:"var(--n-border-color-focus)"})]),f("&:hover",[u("state-border",{borderColor:"var(--n-border-color-hover)"})]),f("&:active",[u("state-border",{borderColor:"var(--n-border-color-pressed)"})]),x("pressed",[u("state-border",{borderColor:"var(--n-border-color-pressed)"})])])]),x("disabled",{backgroundColor:"var(--n-color-disabled)",color:"var(--n-text-color-disabled)"},[u("border",{border:"var(--n-border-disabled)"})]),ee("disabled",[f("&:focus",{backgroundColor:"var(--n-color-focus)",color:"var(--n-text-color-focus)"},[u("state-border",{border:"var(--n-border-focus)"})]),f("&:hover",{backgroundColor:"var(--n-color-hover)",color:"var(--n-text-color-hover)"},[u("state-border",{border:"var(--n-border-hover)"})]),f("&:active",{backgroundColor:"var(--n-color-pressed)",color:"var(--n-text-color-pressed)"},[u("state-border",{border:"var(--n-border-pressed)"})]),x("pressed",{backgroundColor:"var(--n-color-pressed)",color:"var(--n-text-color-pressed)"},[u("state-border",{border:"var(--n-border-pressed)"})])]),x("loading","cursor: wait;"),N("base-wave",`
 pointer-events: none;
 top: 0;
 right: 0;
 bottom: 0;
 left: 0;
 animation-iteration-count: 1;
 animation-duration: var(--n-ripple-duration);
 animation-timing-function: var(--n-bezier-ease-out), var(--n-bezier-ease-out);
 `,[x("active",{zIndex:1,animationName:"button-wave-spread, button-wave-opacity"})]),K&&"MozBoxSizing"in document.createElement("div").style?f("&::moz-focus-inner",{border:0}):null,u("border, state-border",`
 position: absolute;
 left: 0;
 top: 0;
 right: 0;
 bottom: 0;
 border-radius: inherit;
 transition: border-color .3s var(--n-bezier);
 pointer-events: none;
 `),u("border",`
 border: var(--n-border);
 `),u("state-border",`
 border: var(--n-border);
 border-color: #0000;
 z-index: 1;
 `),u("icon",`
 margin: var(--n-icon-margin);
 margin-left: 0;
 height: var(--n-icon-size);
 width: var(--n-icon-size);
 max-width: var(--n-icon-size);
 font-size: var(--n-icon-size);
 position: relative;
 flex-shrink: 0;
 `,[N("icon-slot",`
 height: var(--n-icon-size);
 width: var(--n-icon-size);
 position: absolute;
 left: 0;
 top: 50%;
 transform: translateY(-50%);
 display: flex;
 align-items: center;
 justify-content: center;
 `,[Ee({top:"50%",originalTransform:"translateY(-50%)"})]),qe()]),u("content",`
 display: flex;
 align-items: center;
 flex-wrap: nowrap;
 min-width: 0;
 `,[f("~",[u("icon",{margin:"var(--n-icon-margin)",marginRight:0})])]),x("block",`
 display: flex;
 width: 100%;
 `),x("dashed",[u("border, state-border",{borderStyle:"dashed !important"})]),x("disabled",{cursor:"not-allowed",opacity:"var(--n-opacity-disabled)"})]),f("@keyframes button-wave-spread",{from:{boxShadow:"0 0 0.5px 0 var(--n-ripple-color)"},to:{boxShadow:"0 0 0.5px 4.5px var(--n-ripple-color)"}}),f("@keyframes button-wave-opacity",{from:{opacity:"var(--n-wave-opacity)"},to:{opacity:0}})]),Ze=Object.assign(Object.assign({},de.props),{color:String,textColor:String,text:Boolean,block:Boolean,loading:Boolean,disabled:Boolean,circle:Boolean,size:String,ghost:Boolean,round:Boolean,secondary:Boolean,tertiary:Boolean,quaternary:Boolean,strong:Boolean,focusable:{type:Boolean,default:!0},keyboard:{type:Boolean,default:!0},tag:{type:String,default:"button"},type:{type:String,default:"default"},dashed:Boolean,renderIcon:Function,iconPlacement:{type:String,default:"left"},attrType:{type:String,default:"button"},bordered:{type:Boolean,default:!0},onClick:[Function,Array],nativeFocusBehavior:{type:Boolean,default:!Ye},spinProps:Object}),eo=se({name:"Button",props:Ze,slots:Object,setup(e){const h=H(null),v=H(null),d=H(!1),n=je(()=>!e.quaternary&&!e.tertiary&&!e.secondary&&!e.text&&(!e.color||e.ghost||e.dashed)&&e.bordered),M=ie(Je,{}),{inlineThemeDisabled:E,mergedClsPrefixRef:F,mergedRtlRef:m,mergedComponentPropsRef:P}=De(e),{mergedSizeRef:V}=Qe({},{defaultSize:"medium",mergedSize:t=>{var c,g;const{size:o}=e;if(o)return o;const{size:R}=M;if(R)return R;const{mergedSize:z}=t||{};if(z)return z.value;const I=(g=(c=P==null?void 0:P.value)===null||c===void 0?void 0:c.Button)===null||g===void 0?void 0:g.size;return I||"medium"}}),G=T(()=>e.focusable&&!e.disabled),ce=t=>{var c;G.value||t.preventDefault(),!e.nativeFocusBehavior&&(t.preventDefault(),!e.disabled&&G.value&&((c=h.value)===null||c===void 0||c.focus({preventScroll:!0})))},ue=t=>{var c;if(!e.disabled&&!e.loading){const{onClick:g}=e;g&&Ge(g,t),e.text||(c=v.value)===null||c===void 0||c.play()}},be=t=>{switch(t.key){case"Enter":if(!e.keyboard)return;d.value=!1}},fe=t=>{switch(t.key){case"Enter":if(!e.keyboard||e.loading){t.preventDefault();return}d.value=!0}},he=()=>{d.value=!1},ve=de("Button","-button",Ue,Ve,e,F),pe=_e("Button",m,F),Y=T(()=>{const t=ve.value,{common:{cubicBezierEaseInOut:c,cubicBezierEaseOut:g},self:o}=t,{rippleDuration:R,opacityDisabled:z,fontWeight:I,fontWeightStrong:W}=o,y=V.value,{dashed:Q,type:$,ghost:q,text:C,color:a,round:J,circle:A,textColor:S,secondary:me,tertiary:U,quaternary:ge,strong:ye}=e,xe={"--n-font-weight":ye?W:I};let s={"--n-color":"initial","--n-color-hover":"initial","--n-color-pressed":"initial","--n-color-focus":"initial","--n-color-disabled":"initial","--n-ripple-color":"initial","--n-text-color":"initial","--n-text-color-hover":"initial","--n-text-color-pressed":"initial","--n-text-color-focus":"initial","--n-text-color-disabled":"initial"};const O=$==="tertiary",Z=$==="default",i=O?"default":$;if(C){const l=S||a;s={"--n-color":"#0000","--n-color-hover":"#0000","--n-color-pressed":"#0000","--n-color-focus":"#0000","--n-color-disabled":"#0000","--n-ripple-color":"#0000","--n-text-color":l||o[r("textColorText",i)],"--n-text-color-hover":l?B(l):o[r("textColorTextHover",i)],"--n-text-color-pressed":l?_(l):o[r("textColorTextPressed",i)],"--n-text-color-focus":l?B(l):o[r("textColorTextHover",i)],"--n-text-color-disabled":l||o[r("textColorTextDisabled",i)]}}else if(q||Q){const l=S||a;s={"--n-color":"#0000","--n-color-hover":"#0000","--n-color-pressed":"#0000","--n-color-focus":"#0000","--n-color-disabled":"#0000","--n-ripple-color":a||o[r("rippleColor",i)],"--n-text-color":l||o[r("textColorGhost",i)],"--n-text-color-hover":l?B(l):o[r("textColorGhostHover",i)],"--n-text-color-pressed":l?_(l):o[r("textColorGhostPressed",i)],"--n-text-color-focus":l?B(l):o[r("textColorGhostHover",i)],"--n-text-color-disabled":l||o[r("textColorGhostDisabled",i)]}}else if(me){const l=Z?o.textColor:O?o.textColorTertiary:o[r("color",i)],b=a||l,j=$!=="default"&&$!=="tertiary";s={"--n-color":j?D(b,{alpha:Number(o.colorOpacitySecondary)}):o.colorSecondary,"--n-color-hover":j?D(b,{alpha:Number(o.colorOpacitySecondaryHover)}):o.colorSecondaryHover,"--n-color-pressed":j?D(b,{alpha:Number(o.colorOpacitySecondaryPressed)}):o.colorSecondaryPressed,"--n-color-focus":j?D(b,{alpha:Number(o.colorOpacitySecondaryHover)}):o.colorSecondaryHover,"--n-color-disabled":o.colorSecondary,"--n-ripple-color":"#0000","--n-text-color":b,"--n-text-color-hover":b,"--n-text-color-pressed":b,"--n-text-color-focus":b,"--n-text-color-disabled":b}}else if(U||ge){const l=Z?o.textColor:O?o.textColorTertiary:o[r("color",i)],b=a||l;U?(s["--n-color"]=o.colorTertiary,s["--n-color-hover"]=o.colorTertiaryHover,s["--n-color-pressed"]=o.colorTertiaryPressed,s["--n-color-focus"]=o.colorSecondaryHover,s["--n-color-disabled"]=o.colorTertiary):(s["--n-color"]=o.colorQuaternary,s["--n-color-hover"]=o.colorQuaternaryHover,s["--n-color-pressed"]=o.colorQuaternaryPressed,s["--n-color-focus"]=o.colorQuaternaryHover,s["--n-color-disabled"]=o.colorQuaternary),s["--n-ripple-color"]="#0000",s["--n-text-color"]=b,s["--n-text-color-hover"]=b,s["--n-text-color-pressed"]=b,s["--n-text-color-focus"]=b,s["--n-text-color-disabled"]=b}else s={"--n-color":a||o[r("color",i)],"--n-color-hover":a?B(a):o[r("colorHover",i)],"--n-color-pressed":a?_(a):o[r("colorPressed",i)],"--n-color-focus":a?B(a):o[r("colorFocus",i)],"--n-color-disabled":a||o[r("colorDisabled",i)],"--n-ripple-color":a||o[r("rippleColor",i)],"--n-text-color":S||(a?o.textColorPrimary:O?o.textColorTertiary:o[r("textColor",i)]),"--n-text-color-hover":S||(a?o.textColorHoverPrimary:o[r("textColorHover",i)]),"--n-text-color-pressed":S||(a?o.textColorPressedPrimary:o[r("textColorPressed",i)]),"--n-text-color-focus":S||(a?o.textColorFocusPrimary:o[r("textColorFocus",i)]),"--n-text-color-disabled":S||(a?o.textColorDisabledPrimary:o[r("textColorDisabled",i)])};let L={"--n-border":"initial","--n-border-hover":"initial","--n-border-pressed":"initial","--n-border-focus":"initial","--n-border-disabled":"initial"};C?L={"--n-border":"none","--n-border-hover":"none","--n-border-pressed":"none","--n-border-focus":"none","--n-border-disabled":"none"}:L={"--n-border":o[r("border",i)],"--n-border-hover":o[r("borderHover",i)],"--n-border-pressed":o[r("borderPressed",i)],"--n-border-focus":o[r("borderFocus",i)],"--n-border-disabled":o[r("borderDisabled",i)]};const{[r("height",y)]:X,[r("fontSize",y)]:Ce,[r("padding",y)]:we,[r("paddingRound",y)]:ze,[r("iconSize",y)]:$e,[r("borderRadius",y)]:Se,[r("iconMargin",y)]:Be,waveOpacity:Te}=o,Pe={"--n-width":A&&!C?X:"initial","--n-height":C?"initial":X,"--n-font-size":Ce,"--n-padding":A||C?"initial":J?ze:we,"--n-icon-size":$e,"--n-icon-margin":Be,"--n-border-radius":C?"initial":A||J?X:Se};return Object.assign(Object.assign(Object.assign(Object.assign({"--n-bezier":c,"--n-bezier-ease-out":g,"--n-ripple-duration":R,"--n-opacity-disabled":z,"--n-wave-opacity":Te},xe),s),L),Pe)}),k=E?Ne("button",T(()=>{let t="";const{dashed:c,type:g,ghost:o,text:R,color:z,round:I,circle:W,textColor:y,secondary:Q,tertiary:$,quaternary:q,strong:C}=e;c&&(t+="a"),o&&(t+="b"),R&&(t+="c"),I&&(t+="d"),W&&(t+="e"),Q&&(t+="f"),$&&(t+="g"),q&&(t+="h"),C&&(t+="i"),z&&(t+=`j${re(z)}`),y&&(t+=`k${re(y)}`);const{value:a}=V;return t+=`l${a[0]}`,t+=`m${g[0]}`,t}),Y,e):void 0;return{selfElRef:h,waveElRef:v,mergedClsPrefix:F,mergedFocusable:G,mergedSize:V,showBorder:n,enterPressed:d,rtlEnabled:pe,handleMousedown:ce,handleKeydown:fe,handleBlur:he,handleKeyup:be,handleClick:ue,customColorCssVars:T(()=>{const{color:t}=e;if(!t)return null;const c=B(t);return{"--n-border-color":t,"--n-border-color-hover":c,"--n-border-color-pressed":_(t),"--n-border-color-focus":c,"--n-border-color-disabled":t}}),cssVars:E?void 0:Y,themeClass:k==null?void 0:k.themeClass,onRender:k==null?void 0:k.onRender}},render(){const{mergedClsPrefix:e,tag:h,onRender:v}=this;v==null||v();const d=oe(this.$slots.default,n=>n&&p("span",{class:`${e}-button__content`},n));return p(h,{ref:"selfElRef",class:[this.themeClass,`${e}-button`,`${e}-button--${this.type}-type`,`${e}-button--${this.mergedSize}-type`,this.rtlEnabled&&`${e}-button--rtl`,this.disabled&&`${e}-button--disabled`,this.block&&`${e}-button--block`,this.enterPressed&&`${e}-button--pressed`,!this.text&&this.dashed&&`${e}-button--dashed`,this.color&&`${e}-button--color`,this.secondary&&`${e}-button--secondary`,this.loading&&`${e}-button--loading`,this.ghost&&`${e}-button--ghost`],tabindex:this.mergedFocusable?0:-1,type:this.attrType,style:this.cssVars,disabled:this.disabled,onClick:this.handleClick,onBlur:this.handleBlur,onMousedown:this.handleMousedown,onKeyup:this.handleKeyup,onKeydown:this.handleKeydown},this.iconPlacement==="right"&&d,p(Oe,{width:!0},{default:()=>oe(this.$slots.icon,n=>(this.loading||this.renderIcon||n)&&p("span",{class:`${e}-button__icon`,style:{margin:We(this.$slots.default)?"0":""}},p(Ke,null,{default:()=>this.loading?p(Me,Object.assign({clsPrefix:e,key:"loading",class:`${e}-icon-slot`,strokeWidth:20},this.spinProps)):p("div",{key:"icon",class:`${e}-icon-slot`,role:"none"},this.renderIcon?this.renderIcon():n)})))}),this.iconPlacement==="left"&&d,this.text?null:p(Le,{ref:"waveElRef",clsPrefix:e}),this.showBorder?p("div",{"aria-hidden":!0,class:`${e}-button__border`,style:this.customColorCssVars}):null,this.showBorder?p("div",{"aria-hidden":!0,class:`${e}-button__state-border`,style:this.customColorCssVars}):null)}}),io=eo;export{eo as B,io as X,te as f,Ye as i,Qe as u};
