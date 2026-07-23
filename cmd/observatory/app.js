// winze observatory — fleet dashboard, served by cmd/observatory (go:embed).
// No bundler: JSDoc-typed vanilla JS, checked with tsc (jsconfig checkJs) + biome.
/**
 * @typedef {{l?:string, c:number, d:number}} GraphNode
 * @typedef {{name:string, kind:string, tier:number, entities:number, claims:number, nodes:GraphNode[], edges:[number,number][], efam:number[], trips:[number,number,number][], spent:number, cap:number, last_activity:number}} Instance
 * @typedef {{instances:Instance[]}} Fleet
 * @typedef {{ts:number, name:string, kind:string, payload?:Record<string, any>}} Ev
 * @typedef {{x:number, y:number}} Pt
 * @typedef {{inst:Instance, fig:HTMLElement, cv:HTMLCanvasElement, ctx:CanvasRenderingContext2D, pos:Pt[], statusEl:(Element|null), offset:number, phaseFlash:({phase:string, color:string, until:number}|null), w?:number, h?:number, rect?:DOMRect}} Cell
 * @typedef {{a:Cell, b:Cell, start:number, removeAt:(number|null), key:string}} Meld
 */

/** @type {Fleet} */
let FLEET = { instances: [] };

const DPR = Math.min(window.devicePixelRatio || 1, 2);
const reduce = matchMedia('(prefers-reduced-motion:reduce)').matches;
function getVar(n){return getComputedStyle(document.documentElement).getPropertyValue(n).trim();}
// mineral specimens by community — earthy assay tones, USGS-adjacent, never neon
const MINERAL = ['#4f9e6b','#3f6ea5','#c0503a','#c39a3e','#8a63a8','#3f9e93','#b8863f','#8a94a0','#b5533a','#9c7bc2'];
// colour for a real metabolism phase that actually ran (event-driven pulse)
const PHASE_COLOR = {sense:'#4aa3ff',resolve:'#5cc8e8',ingest:'#43e8d4',trip:'#9a72c4',dream:'#5ce8a0',calibrate:'#d69a44',bias:'#c0503a'};
function phaseColor(p){ for(const k in PHASE_COLOR){ if(p.indexOf(k)===0) return PHASE_COLOR[k]; } return getVar('--brass'); }
/** @type {Record<string, Cell>} */
let cellByName = {};
/** @type {Meld[]} */
let melds = [];
let clock = 0;

document.getElementById('key').innerHTML =
  `<span><i class="sw" style="border-top-color:${getVar('--brass')}"></i>typed claim</span>`+
  `<span><i class="sw" style="border-top-color:${getVar('--copper')};border-top-style:dashed"></i>latent link</span>`+
  `<span><i class="sw" style="border-top-color:${getVar('--hematite')}"></i>dispute</span>`+
  `<span><i class="sw" style="border-top-color:${getVar('--amethyst')};border-top-style:dashed"></i>conjecture</span>`+
  `<span><i class="dot" style="background:${getVar('--warm')}"></i>active recently</span>`+
  `<span><i class="dot" style="background:${getVar('--cold')}"></i>idle</span>`;

// ---- layout: settle once, then only decoration + real signals animate ----
function settle(nodes, edges){
  const n=nodes.length;
  const P=nodes.map((_,i)=>({x:Math.cos(i*2.399)*0.4+0.5+Math.random()*.02, y:Math.sin(i*2.399)*0.4+0.5+Math.random()*.02, vx:0,vy:0}));
  const it=n>150?260:180;
  for(let s=0;s<it;s++){
    const k=0.0009, rep=0.00028;
    for(let i=0;i<n;i++){ let fx=0,fy=0;
      for(let j=0;j<n;j++){ if(i===j)continue; const dx=P[i].x-P[j].x, dy=P[i].y-P[j].y, d2=dx*dx+dy*dy+1e-4; const f=rep/d2; fx+=dx*f; fy+=dy*f; }
      fx+=(0.5-P[i].x)*0.004; fy+=(0.5-P[i].y)*0.004;
      P[i].vx=(P[i].vx+fx)*0.86; P[i].vy=(P[i].vy+fy)*0.86;
    }
    for(const [a,b] of edges){ const dx=P[b].x-P[a].x, dy=P[b].y-P[a].y; P[a].vx+=dx*k; P[a].vy+=dy*k; P[b].vx-=dx*k; P[b].vy-=dy*k; }
    for(const p of P){ p.x+=p.vx; p.y+=p.vy; }
  }
  const xs=P.map(p=>p.x), ys=P.map(p=>p.y);
  const nx=Math.min(...xs),Xx=Math.max(...xs),ny=Math.min(...ys),Xy=Math.max(...ys);
  for(const p of P){ p.x=0.07+0.86*(p.x-nx)/(Xx-nx||1); p.y=0.11+0.80*(p.y-ny)/(Xy-ny||1); }
  return P;
}
function scatter(nodes){ return nodes.map((_,i)=>{ const a=i*2.399, r=0.08+0.4*Math.sqrt((i+1)/nodes.length); return {x:0.5+Math.cos(a)*r, y:0.52+Math.sin(a)*r}; }); }

// ---- real recency signal ----
function warmth(inst){ // 1 = freshly metabolized, → 0.18 as it goes cold; memory stores read dormant
  if(!inst.last_activity) return 0.24;
  const ageH=(Date.now()/1000 - inst.last_activity)/3600;
  return Math.max(0.18, Math.min(1, 1 - ageH/48));
}
function agoText(unix){ if(!unix) return null; const s=Date.now()/1000-unix;
  if(s<3600) return Math.round(s/60)+'m ago'; if(s<86400) return Math.round(s/3600)+'h ago'; return Math.round(s/86400)+'d ago'; }
function statusText(inst){
  if(inst.kind==='memory') return 'memory store';
  const a=agoText(inst.last_activity);
  return a ? 'last cycle '+a : 'never run';
}

// ---- cells ----
const fleetEl = document.getElementById('fleet');
/** @type {Cell[]} */
let cells = [];
function buildCells(){
  fleetEl.innerHTML='';
  cells = FLEET.instances.map((inst, idx)=>{
    const fig=document.createElement('figure');
    fig.className='cell'+(idx===0?' hero':'');
    fig.innerHTML=`<canvas role="img" aria-label="${inst.name}: ${inst.entities} entities, ${inst.claims} claims, ${inst.kind}"></canvas>
      <span class="tick tl"></span><span class="tick tr"></span><span class="tick bl"></span><span class="tick br"></span>
      <figcaption class="cell-hd"><span class="cell-name">${inst.name}</span><span class="cell-kind">${inst.kind}</span><span class="cell-status">${statusText(inst)}</span></figcaption>
      <div class="cell-tel"><span><b>${inst.entities}</b> entities</span><span><b>${inst.claims}</b> claims</span>
      <span>budget <b>${inst.spent||0}</b>/<b>${inst.cap||300}</b>¢</span><span class="chip">tier ${inst.tier}</span></div>`;
    fleetEl.appendChild(fig);
    const cv=fig.querySelector('canvas'), ctx=cv.getContext('2d');
    const hasEdges = inst.edges?.length;
    const pos = hasEdges ? settle(inst.nodes, inst.edges) : scatter(inst.nodes);
    return {inst,fig,cv,ctx,pos,statusEl:fig.querySelector('.cell-status'),offset:Math.random()*10,phaseFlash:null};
  });
  cellByName = Object.fromEntries(cells.map(c=>[c.inst.name, c]));
  recomputeAgg();
  fitAll();
  if(reduce) drawStatic();
}
function recomputeAgg(){
  const sum=k=>FLEET.instances.reduce((s,i)=>s+(i[k]||0),0);
  document.getElementById('a-inst').textContent=String(FLEET.instances.length);
  document.getElementById('a-ent').textContent=sum('entities');
  document.getElementById('a-cl').textContent=sum('claims');
  document.getElementById('a-spend').textContent=sum('spent');
  // honest fleet state: newest last_activity across workings
  const newest=Math.max(0,...FLEET.instances.map(i=>i.last_activity||0));
  const a=agoText(newest);
  document.getElementById('a-state').textContent = a ? 'last cycle '+a : 'idle';
}

// ---- resize ----
function fitCanvas(cell){ const r=cell.cv.getBoundingClientRect(); cell.cv.width=Math.max(1,r.width*DPR); cell.cv.height=Math.max(1,r.height*DPR); cell.w=r.width; cell.h=r.height; cell.rect=r; }
const meldCv=/** @type {HTMLCanvasElement} */(document.getElementById('meld-overlay'));
const meldCtx=meldCv.getContext('2d');
function fitAll(){ cells.forEach(fitCanvas); meldCv.width=innerWidth*DPR; meldCv.height=innerHeight*DPR; }
addEventListener('resize', fitAll);

function withAlpha(hex,a){ const n=parseInt(hex.slice(1),16); return `rgba(${n>>16&255},${n>>8&255},${n&255},${a})`; }

function drawCell(cell, time){
  const {ctx,inst,pos}=cell; const W=cell.w*DPR, H=cell.h*DPR;
  ctx.clearRect(0,0,W,H);
  const w=warmth(inst);                          // REAL recency, 0.18..1
  const br=reduce?1:1+0.010*Math.sin(time*0.6+cell.offset);
  const X=p=>(0.5+(p.x-0.5)*br)*W, Y=p=>(0.5+(p.y-0.5)*br)*H;

  // veins (typed claim edges) + latent seams (wikilinks) + faults (disputes)
  if(inst.edges) for(let e=0;e<inst.edges.length;e++){
    const [a,b]=inst.edges[e]; const fam=inst.efam?inst.efam[e]:0;
    if(fam===3){ ctx.setLineDash([]); ctx.strokeStyle=withAlpha(getVar('--hematite'), (0.22+0.16*(0.5+0.5*Math.sin(time*2.2+e)))*(0.5+0.5*w)); ctx.lineWidth=1.1*DPR; }
    else if(fam===7){ ctx.setLineDash([2.5*DPR,4*DPR]); ctx.strokeStyle=withAlpha(getVar('--copper'), 0.16*(0.5+0.5*w)); ctx.lineWidth=0.7*DPR; }
    else { ctx.setLineDash([]); ctx.strokeStyle=withAlpha(getVar('--brass'), 0.15*(0.5+0.5*w)); ctx.lineWidth=0.7*DPR; }
    ctx.beginPath(); ctx.moveTo(X(pos[a]),Y(pos[a])); ctx.lineTo(X(pos[b]),Y(pos[b])); ctx.stroke();
  }
  ctx.setLineDash([]);

  // prospects — REAL trip conjectures, shown continuously as faint pulsing prospect veins
  if(inst.trips) for(let t=0;t<inst.trips.length;t++){
    const [a,b,sc]=inst.trips[t]; if(a>=pos.length||b>=pos.length) continue;
    const pulse=0.18+0.22*(0.5+0.5*Math.sin(time*1.1+t*1.3));
    const ax=X(pos[a]),ay=Y(pos[a]),bx=X(pos[b]),by=Y(pos[b]);
    const mx=(ax+bx)/2+(ay-by)*0.22, my=(ay+by)/2+(bx-ax)*0.22;
    ctx.setLineDash([2*DPR,5*DPR]); ctx.strokeStyle=withAlpha(getVar('--amethyst'), pulse*(sc>=4?1:0.7));
    ctx.lineWidth=1.1*DPR; ctx.beginPath(); ctx.moveTo(ax,ay); ctx.quadraticCurveTo(mx,my,bx,by); ctx.stroke();
  }
  ctx.setLineDash([]);

  // specimens (nodes) — mineral by community, size by degree, brightness by REAL recency
  for(let i=0;i<pos.length;i++){
    const nd=inst.nodes[i]; const col=MINERAL[nd.c%MINERAL.length]; const deg=nd.d||0;
    const rad=(1.5+Math.min(deg,24)*0.16)*DPR; const x=X(pos[i]),y=Y(pos[i]);
    const shimmer=reduce?0.8:0.62+0.38*Math.sin(time*1.0+i*0.6);
    const thin=deg<=1?0.5:1;
    const lum=(0.35+0.65*w)*thin;                // recency drives luminance
    const g=ctx.createRadialGradient(x,y,0,x,y,rad*3.2);
    g.addColorStop(0,withAlpha(col, 0.5*lum*shimmer)); g.addColorStop(1,withAlpha(col,0));
    ctx.fillStyle=g; ctx.beginPath(); ctx.arc(x,y,rad*3.2,0,7); ctx.fill();
    ctx.fillStyle=withAlpha(col, 0.85*lum); ctx.beginPath(); ctx.arc(x,y,rad,0,7); ctx.fill();
  }

  // recency temperature wash: warm lamplight when freshly worked, cold when dormant
  const temp = w>0.5 ? getVar('--warm') : getVar('--cold');
  const vg=ctx.createRadialGradient(W*0.5,H*0.5,0,W*0.5,H*0.5,Math.max(W,H)*0.75);
  vg.addColorStop(0,withAlpha(temp, 0.018+0.03*Math.abs(w-0.5))); vg.addColorStop(1,'rgba(0,0,0,0)');
  ctx.fillStyle=vg; ctx.fillRect(0,0,W,H);

  // real phase pulse — only when a metabolism phase actually fired (event-driven)
  let running=null;
  if(cell.phaseFlash && clock < cell.phaseFlash.until){
    const pf=cell.phaseFlash, t=1-(pf.until-clock)/2.5, a=Math.sin(Math.min(1,Math.max(0,t))*Math.PI)*0.5;
    const pvg=ctx.createRadialGradient(W*0.5,H*0.12,0,W*0.5,H*0.12,Math.max(W,H));
    pvg.addColorStop(0,withAlpha(pf.color,0.16*a)); pvg.addColorStop(1,'rgba(0,0,0,0)');
    ctx.fillStyle=pvg; ctx.fillRect(0,0,W,H);
    running=pf.phase;
  } else if(cell.phaseFlash){ cell.phaseFlash=null; }

  if(cell.statusEl) cell.statusEl.textContent = running ? ('running · '+running) : statusText(inst);
}
function drawStatic(){ for(const c of cells) drawCell(c,2); }

// ---- real meld bridges (driven by winze-meld events, not a timer) ----
function drawMelds(time){
  meldCtx.clearRect(0,0,meldCv.width,meldCv.height);
  const lab=document.getElementById('meld-label');
  melds = melds.filter(m=>!(m.removeAt!==null && time>m.removeAt+1));
  if(!melds.length){ lab.style.opacity='0'; return; }
  const col=getVar('--warm'); let labeled=false;
  for(const m of melds){
    const ra=m.a.rect, rb=m.b.rect; if(!ra||!rb) continue;
    const grow=Math.min(1,(time-m.start)/0.8);
    const fade=m.removeAt!==null ? Math.max(0,1-(time-m.removeAt)/1) : 1;
    const env=grow*fade; if(env<=0) continue;
    const ax=(ra.left+ra.width*0.5)*DPR, ay=(ra.top+ra.height*0.5)*DPR, bx=(rb.left+rb.width*0.5)*DPR, by=(rb.top+rb.height*0.5)*DPR;
    const mx=(ax+bx)/2, my=(ay+by)/2-80*DPR;
    meldCtx.lineWidth=2.2*DPR; meldCtx.strokeStyle=withAlpha(col,0.5*env); meldCtx.shadowColor=col; meldCtx.shadowBlur=16*DPR;
    meldCtx.beginPath(); meldCtx.moveTo(ax,ay); meldCtx.quadraticCurveTo(mx,my,bx,by); meldCtx.stroke(); meldCtx.shadowBlur=0;
    for(let k=0;k<12;k++){ let u=((time*0.5)+(k/12))%1; if(k%2)u=1-u;
      const x=(1-u)*(1-u)*ax+2*(1-u)*u*mx+u*u*bx, y=(1-u)*(1-u)*ay+2*(1-u)*u*my+u*u*by;
      meldCtx.fillStyle=withAlpha(col,env*(0.4+0.5*Math.sin(u*Math.PI))); meldCtx.beginPath(); meldCtx.arc(x,y,2.2*DPR,0,7); meldCtx.fill(); }
    for(const [x,y] of [[ax,ay],[bx,by]]){ const g=meldCtx.createRadialGradient(x,y,0,x,y,50*DPR); g.addColorStop(0,withAlpha(col,0.35*env)); g.addColorStop(1,withAlpha(col,0)); meldCtx.fillStyle=g; meldCtx.beginPath(); meldCtx.arc(x,y,50*DPR,0,7); meldCtx.fill(); }
    if(!labeled){ lab.textContent='meld · '+m.a.inst.name+' ⇄ '+m.b.inst.name; lab.style.left=(mx/DPR)+'px'; lab.style.top=(my/DPR-14)+'px'; lab.style.transform='translate(-50%,-100%)'; lab.style.opacity=String(env); labeled=true; }
  }
  if(!labeled) lab.style.opacity='0';
}

// ---- clock + live loader ----
const clockEl=document.getElementById('clock'), liveEl=document.getElementById('live');
function updateClock(){ const d=new Date(); clockEl.textContent=[d.getHours(),d.getMinutes(),d.getSeconds()].map(x=>String(x).padStart(2,'0')).join(':'); }
function sig(f){ return f.instances.map(i=>`${i.name}:${i.entities}:${i.claims}:${(i.edges||[]).length}`).join('|'); }
async function refresh(){
  try{ const r=await fetch('/api/fleet.json',{cache:'no-store'}); if(!r.ok) throw 0;
    const data=await r.json(); if(data.instances?.length){
      if(sig(data)!==sig(FLEET)){ FLEET=data; buildCells(); } else { FLEET=data; recomputeAgg(); }
      liveEl.textContent='live'; liveEl.classList.remove('stale'); }
  }catch(_e){ liveEl.textContent='snapshot'; liveEl.classList.add('stale'); }
}

buildCells();
refresh();
setInterval(refresh, 20000);
setInterval(updateClock, 1000); updateClock();

// ---- consume the real fleet event stream ----
const seenEv=new Set(); let evInit=false;
function handleEvent(ev){
  const c=cellByName[ev.name];
  if(ev.kind==='phase' && ev.payload && ev.payload.decision==='allow'){
    if(c) c.phaseFlash={phase:ev.payload.phase, color:phaseColor(ev.payload.phase), until:clock+2.5};
  } else if(ev.kind==='cycle_start'){ if(c) c.fig.classList.add('active'); }
  else if(ev.kind==='cycle_end'){ if(c) c.fig.classList.remove('active'); }
  else if(ev.kind==='meld'){
    const cs=((ev.payload?.stores)||[]).map(n=>cellByName[n]).filter(Boolean);
    const key=(ev.payload?.dir)||'';
    for(let i=0;i<cs.length;i++) for(let j=i+1;j<cs.length;j++) melds.push({a:cs[i],b:cs[j],start:clock,removeAt:null,key});
  } else if(ev.kind==='unmeld'){
    const key=(ev.payload?.dir)||'';
    for(const m of melds) if(m.key===key && m.removeAt===null) m.removeAt=Math.max(clock, m.start+3);
  }
}
async function pollEvents(){
  try{ const r=await fetch('/api/events.json',{cache:'no-store'}); if(!r.ok) return;
    const evs=(await r.json()).events||[];
    for(const ev of evs){ const sig=ev.ts+'|'+ev.kind+'|'+ev.name+'|'+JSON.stringify(ev.payload||{});
      if(seenEv.has(sig)) continue; seenEv.add(sig); if(evInit) handleEvent(ev); }
    evInit=true;
    if(seenEv.size>5000){ seenEv.clear(); evInit=false; } // don't replay history on reseed
  }catch{}
}
setInterval(pollEvents, 3000); pollEvents();

let t0=null, lastDraw=0;
function frame(ts){
  requestAnimationFrame(frame);
  if(ts-lastDraw < 33) return;   // ~30fps for an always-on sheet
  lastDraw=ts;
  if(t0===null)t0=ts; clock=(ts-t0)/1000;
  for(const c of cells) drawCell(c,clock);
  drawMelds(clock);
}
if(reduce) drawStatic(); else requestAnimationFrame(frame);
