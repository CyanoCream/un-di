/* Undangan Digital runtime (ES2017, tanpa dependensi) */
(function(){
'use strict';
var d=document,R=d.documentElement,B=d.body,W=window;
var $=(s,c)=>(c||d).querySelector(s),$$=(s,c)=>[].slice.call((c||d).querySelectorAll(s));
var at=(e,a)=>e.getAttribute(a),sa=(e,a,v)=>e.setAttribute(a,v);
var on=(e,t,f,o)=>e.addEventListener(t,f,o);
var cl=(e,c,v)=>e.classList.toggle(c,v),has=(e,c)=>e.classList.contains(c);
var el=(t,c,x)=>{var e=d.createElement(t);if(c)e.className=c;if(x!=null)e.textContent=x;return e};
var preview=at(B,'data-preview')==='1',NET='Gagal terhubung. Periksa koneksi Anda.',NM='Mohon isi nama (minimal 2 karakter).',API='/api/v1/public/invitations/',IO='IntersectionObserver' in W;
var reduce=matchMedia('(prefers-reduced-motion: reduce)').matches;
cl(R,'rt',true);
if(IO&&!reduce)cl(R,'can-reveal',true);
var tEl,tT;
function toast(m){
if(!tEl){tEl=el('div','toast');sa(tEl,'role','status');B.append(tEl)}
tEl.textContent=m;cl(tEl,'is-shown',true);
clearTimeout(tT);tT=setTimeout(()=>cl(tEl,'is-shown',false),2400);
}
// musik (mulai dari gestur klik)
var au=$('[data-music]'),tog=$('[data-music-toggle]'),resume=false;
function play(){
if(!au)return;
var st=parseFloat(at(au,'data-start'))||0;
if(st>0&&!au._s){
au._s=1;
var seek=()=>{try{if(au.currentTime<st)au.currentTime=st}catch(e){}};
au.readyState>0?seek():on(au,'loadedmetadata',seek,{once:true});
}
au.play().catch(()=>mUI(false));
}
function mUI(v){
if(!tog)return;
cl(tog,'is-playing',v);sa(tog,'aria-pressed',v);sa(tog,'aria-label',v?'Jeda musik':'Putar musik');
}
if(au){
on(au,'play',()=>mUI(true));on(au,'pause',()=>mUI(false));
if(tog)on(tog,'click',()=>{resume=false;au.paused?play():au.pause()});
on(d,'visibilitychange',()=>{
if(d.hidden){resume=!au.paused;if(resume)au.pause()}
else if(resume){resume=false;play()}
});
}
// cover
var cover=$('[data-cover]');
function opened(){
cl(R,'is-opened',true);
var dk=$('[data-dock]');
if(tog)tog.hidden=false;
if(dk)dk.hidden=false;
var els=$$('[data-reveal]');
if(!has(R,'can-reveal'))return els.forEach(x=>cl(x,'is-visible',true));
var io=new IntersectionObserver(es=>es.forEach(x=>{
if(x.isIntersecting){cl(x.target,'is-visible',true);io.unobserve(x.target)}
}),{rootMargin:'0px 0px -6% 0px',threshold:.06});
els.forEach(x=>io.observe(x));
}
if(cover){
R.classList.add('has-cover','is-locked');
if('scrollRestoration' in history)history.scrollRestoration='manual';
scrollTo(0,0);
$$('[data-open]',cover).forEach(b=>on(b,'click',e=>{
e.preventDefault();
if(has(R,'is-opened'))return;
play();
cl(R,'is-locked',false);
scrollTo({top:0,behavior:'instant'});
sa(cover,'aria-hidden','true');
opened();
setTimeout(()=>{cover.hidden=true},reduce?0:1100);
var m=$('main');
if(m){sa(m,'tabindex','-1');m.focus({preventScroll:true})}
}));
}else opened();
// hitung mundur
$$('[data-countdown]').forEach(box=>{
var s0=Date.parse(at(box,'data-countdown')),end=Date.parse(at(box,'data-countdown-end')||''),P={},tm;
if(isNaN(s0))return;
if(!(end>s0))end=s0+72e5;
$$('[data-cd]',box).forEach(n=>{P[at(n,'data-cd')]=n});
var done=$('[data-cd-done]',box);
var tick=()=>{
var now=Date.now(),s=Math.floor((s0-now)/1000);
if(s<=0){
cl(box,'is-done',true);
if(done){done.textContent=now<end?'Acara sedang berlangsung':'Acara telah berlangsung';done.hidden=false}
if(now>=end)clearInterval(tm);
return;
}
var v={d:s/86400,h:s%86400/3600,m:s%3600/60,s:s%60};
for(var k in P)P[k].textContent=String(v[k]|0).padStart(2,'0');
};
tm=setInterval(tick,1000);tick();
});
// salin ke clipboard
async function copy(t){
try{if(navigator.clipboard){await navigator.clipboard.writeText(t);return true}}catch(e){}
var ta=el('textarea'),ok=false;
ta.value=t;sa(ta,'readonly','');ta.style.cssText='position:fixed;top:0;opacity:0';
B.append(ta);ta.select();
try{ok=d.execCommand('copy')}catch(e){}
ta.remove();return ok;
}
on(d,'click',async e=>{
var b=e.target.closest&&e.target.closest('[data-copy]');
if(!b)return;
var ok=await copy(at(b,'data-copy'));
toast(ok?at(b,'data-copy-msg')||'Berhasil disalin':'Gagal menyalin');
if(ok){cl(b,'is-copied',true);setTimeout(()=>cl(b,'is-copied',false),1600)}
});
// ucapan & RSVP: teks pengguna hanya via textContent
var form=$('[data-wishes-form]'),wb=$('[data-wishes]'),host=form||wb,inv=host&&at(host,'data-invitation');
var api=inv?API+encodeURIComponent(inv)+'/wishes':'';
var LBL={hadir:'Hadir',tidak:'Tidak Hadir',ragu:'Masih Ragu'};
var UN=[[31536e3,'tahun'],[2592e3,'bulan'],[604800,'minggu'],[86400,'hari'],[3600,'jam'],[60,'menit']];
function ago(iso){
var s=(Date.now()-Date.parse(iso))/1000;
if(isNaN(s))return '';
for(var u of UN)if(s>=u[0])return (s/u[0]|0)+' '+u[1]+' lalu';
return 'baru saja';
}
function wish(w){
var name=String(w.name||'Tamu'),a=LBL[w.attendance]?w.attendance:'';
var li=el('li','wish'),h=el('div','wish__head'),who=el('div','wish__who'),t=el('time','wish__time',ago(w.created_at));
h.append(el('span','wish__avatar',(Array.from(name.trim())[0]||'?').toUpperCase()));
who.append(el('strong','wish__name',name));
if(w.created_at)sa(t,'datetime',w.created_at);
who.append(t);h.append(who);
if(a)h.append(el('span','wish__badge wish__badge--'+a,LBL[a]));
li.append(h);
if(w.message)li.append(el('p','wish__msg',w.message));
return li;
}
var page=1,cnt=0,busy=false;
async function load(reset){
if(!wb||!api||busy)return;
busy=true;
var q=s=>$('[data-wishes-'+s+']',wb),list=q('list'),more=q('more'),empty=q('empty');
if(reset)page=1;
try{
var r=await fetch(api+'?page='+page,{headers:{Accept:'application/json'}});
if(!r.ok)throw r.status;
var j=await r.json(),items=j.items||[],st=j.stats||{};
if(reset){list.textContent='';cnt=0}
items.forEach(w=>list.append(wish(w)));
cnt+=items.length;page++;
$$('[data-stat]',wb).forEach(n=>{var k=at(n,'data-stat');n.textContent=(k==='total'?j.total:st[k])||0});
if(more)more.hidden=!items.length||cnt>=(j.total||0);
}catch(e){if(more)more.hidden=true}
if(empty)empty.hidden=cnt>0;
busy=false;
}
if(wb){
var mb=$('[data-wishes-more]',wb);
if(mb)on(mb,'click',()=>load());
if(IO){var wio=new IntersectionObserver(es=>{if(es[0].isIntersecting){wio.disconnect();load(1)}},{rootMargin:'600px 0px'});wio.observe(wb)}
else load(1);
}
if(form){
var stEl=$('[data-form-status]',form),btn=$('[type=submit]',form),pax=$('[data-pax-field]',form);
var val=n=>{var x=$('[name="'+n+'"]',form);return x?x.value.trim():''};
var say=(m,ok)=>{
if(!stEl)return toast(m);
stEl.textContent=m;stEl.hidden=false;stEl.className='rsvp__status is-'+(ok?'ok':'error');
};
on(form,'change',e=>{if(e.target.name==='attendance'&&pax)pax.hidden=e.target.value==='tidak'});
on(form,'submit',async e=>{
e.preventDefault();
var a=$('[name=attendance]:checked',form);
var data={name:val('name'),attendance:a?a.value:'',pax:parseInt(val('pax'),10)||1,message:val('message'),guest_code:val('guest_code')};
if(data.name.length<2)return say(NM);
if(!data.attendance)return say('Pilih konfirmasi kehadiran.');
if(data.attendance==='tidak')data.pax=1;
if(preview||!api)return say('Mode pratinjau: ucapan tidak dikirim.',true);
var label=btn&&btn.textContent;
if(btn){btn.disabled=true;btn.textContent='Mengirim…'}
try{
var r=await fetch(api,{method:'POST',headers:{'Content-Type':'application/json',Accept:'application/json','X-Requested-With':'fetch'},body:JSON.stringify(data)});
if(r.ok){
$('[name=message]',form).value='';
say('Terima kasih! Ucapan Anda telah terkirim.',true);
load(1);
}else{
var m='Gagal mengirim ucapan. Silakan coba lagi.';
try{var j=await r.json();if(j.error&&j.error.message)m=j.error.message}catch(x){}
say(m);
}
}catch(err){say(NET)}
if(btn){btn.disabled=false;btn.textContent=label}
});
}
var img=s=>new Promise((ok,no)=>{var i=new Image();i.onload=()=>ok(i);i.onerror=no;i.src=s});
var canvas=(w,h)=>{var c=el('canvas');c.width=w;c.height=h;return [c,c.getContext('2d')]};
// tiket check-in → PNG
var tk=$('[data-ticket]');
if(tk)on($('[data-ticket-save]',tk),'click',async()=>{
var g=k=>{var n=$('[data-tk='+k+']',tk);return n?n.textContent.trim():''},q=$('svg.qr',tk);
var cs=getComputedStyle(tk),cv=(k,f)=>cs.getPropertyValue(k).trim()||f;
var ac=cv('--accent','#555'),fd=cv('--font-display','serif'),fb=cv('--font-body','sans-serif');
try{
if(d.fonts)await d.fonts.ready;
var im=await img('data:image/svg+xml,'+encodeURIComponent(q.outerHTML));
var w=600,y=84,[c,x]=canvas(1200,2400);
x.scale(2,2);x.fillStyle='#fff';x.fillRect(0,0,w,1200);x.textAlign='center';
var T=(t,px,f,col,lh)=>{
if(!t)return;
x.font=px+'px '+f;x.fillStyle=col;
var ln='';
t.split(' ').forEach(s=>{var z=ln?ln+' '+s:s;if(ln&&x.measureText(z).width>w-110){x.fillText(ln,w/2,y);y+=lh;ln=s}else ln=z});
x.fillText(ln,w/2,y);y+=lh;
};
T(g('label').toUpperCase(),15,fb,ac,52);
T(g('event'),42,fd,'#111',38);
T(g('date'),18,fb,'#666',34);
x.fillStyle=ac;x.fillRect(w/2-30,y-12,60,2);y+=38;
T(g('name'),30,fd,'#111',36);
T(g('pax'),17,fb,'#555',30);
x.drawImage(im,150,y-12,300,300);y+=340;
T('KODE UNDANGAN',13,fb,'#777',50);
T(g('code').split('').join(' '),'700 46','monospace','#111',52);
T(g('hint'),16,fb,'#555',24);
var h=y+30,[o,z]=canvas(1200,h*2);
z.drawImage(c,0,0);z.scale(2,2);z.strokeStyle=ac;z.lineWidth=3;z.strokeRect(14,14,w-28,h-28);z.lineWidth=1;z.strokeRect(24,24,w-48,h-48);
o.toBlob(b=>{
var a=el('a');a.href=URL.createObjectURL(b);a.download='tiket-'+g('code')+'.png';
B.append(a);a.click();a.remove();setTimeout(()=>URL.revokeObjectURL(a.href),9e3);toast('Tiket disimpan');
},'image/png');
}catch(e){toast('Gagal menyimpan tiket')}
});
// konfirmasi hadiah (multipart + kompresi foto)
var F=$('[data-gift-form]');
if(F){
var gb=$('[data-gift-open]'),gs=$('[data-gift-status]',F),S=$('[type=submit]',F),fi=$('[data-gift-file]',F),pv=$('[data-gift-preview]',F),fn=$('[data-gift-file-name]',F),fn0=fn.textContent,amt=$('[data-rupiah]',F),pu;
var HE='Format foto tidak didukung. Silakan kirim tangkapan layar (screenshot) atau foto JPG.';
var V=n=>{var x=F.elements[n];return x&&x.value?x.value.trim():''};
var Y=(m,ok)=>{gs.textContent=m;gs.hidden=false;gs.className='rsvp__status is-'+(ok?'ok':'error')};
var tt=()=>$$('[data-gift-transfer]',F).forEach(f=>{f.hidden=V('type')=='kado'});
var pick=()=>{
var f=fi.files[0];
if(pu)URL.revokeObjectURL(pu);
pu=0;pv.hidden=gs.hidden=true;fn.textContent=f?f.name:fn0;
if(f){pu=URL.createObjectURL(f);pv.onload=()=>{pv.hidden=false};pv.onerror=()=>Y(HE);pv.src=pu}
};
var prep=async f=>{
if(/^image\/(jpeg|png|webp)$/.test(f.type)&&f.size<=1048576)return f;
var u=URL.createObjectURL(f),i,bl=t=>new Promise(r=>c.toBlob(r,t,.8));
try{i=await img(u)}catch(e){throw HE}finally{URL.revokeObjectURL(u)}
var W0=i.naturalWidth,H0=i.naturalHeight,s=Math.min(1,1600/Math.max(W0,H0)),[c,x]=canvas(Math.round(W0*s),Math.round(H0*s));
x.fillStyle='#fff';x.fillRect(0,0,c.width,c.height);x.drawImage(i,0,0,c.width,c.height);
var b=await bl('image/webp');
if(!b||b.type!='image/webp')b=await bl('image/jpeg');
if(!b)throw HE;
return b;
};
if(gb)on(gb,'click',()=>{var o=F.hidden;F.hidden=!o;sa(gb,'aria-expanded',o);cl(gb,'is-open',o);if(o)F.elements.name.focus()});
if(amt)on(amt,'input',()=>{amt.value=amt.value.replace(/\D/g,'').replace(/^0+/,'').slice(0,12).replace(/\B(?=(\d{3})+$)/g,'.')});
on(F,'change',e=>{if(e.target.name=='type')tt();if(e.target==fi)pick()});
on(F,'submit',async e=>{
e.preventDefault();
var f=fi.files[0],k=V('type')=='kado',lb=S.textContent;
if(V('name').length<2)return Y(NM);
if(!f)return Y('Mohon unggah foto bukti transfer / kado.');
S.disabled=true;S.textContent='Memproses foto…';
try{
var b=await prep(f);
if(b.size>5242880)throw 'Ukuran foto maksimal 5 MB. Silakan pilih foto lain.';
if(preview)throw 0;
var fd=new FormData();
[['name',V('name')],['type',k?'kado':'transfer'],['account_label',k?'':V('account_label')],['amount',k?'':V('amount').replace(/\D/g,'')],['message',V('message')],['guest_code',V('guest_code')]].forEach(p=>fd.append(p[0],p[1]));
fd.append('file',b,b.name||'bukti.'+(b.type=='image/webp'?'webp':'jpg'));
S.textContent='Mengirim…';
var r=await new Promise((ok,no)=>{
var x=new XMLHttpRequest();
x.open('POST',API+encodeURIComponent(at(F,'data-invitation'))+'/gifts');
x.setRequestHeader('X-Requested-With','fetch');
x.upload.onprogress=p=>{if(p.lengthComputable)S.textContent='Mengunggah '+Math.round(p.loaded/p.total*100)+'%'};
x.onload=()=>ok(x);x.onerror=()=>no(NET);
x.send(fd);
});
if(r.status<300){F.reset();pick();tt();Y('Terima kasih, konfirmasi hadiah terkirim 🙏',1)}
else{var m='Gagal mengirim konfirmasi. Silakan coba lagi.';try{m=JSON.parse(r.responseText).error.message||m}catch(x){}Y(m)}
}catch(err){err==0?Y('Mode pratinjau: konfirmasi tidak dikirim.',1):Y(typeof err=='string'?err:'Gagal memproses foto. Silakan coba lagi.')}
S.disabled=false;S.textContent=lb;
});
}
// lightbox galeri
var lb,lbImg,lbN,L=[],I=0,from;
function lbBuild(){
lb=el('div','lightbox');lbImg=el('img','lightbox__img');lbN=el('p','lightbox__count');
sa(lb,'role','dialog');sa(lb,'aria-modal','true');sa(lb,'aria-label','Galeri foto');
lb.append(lbImg,lbN);
[['prev','Foto sebelumnya','‹',()=>go(-1)],['next','Foto berikutnya','›',()=>go(1)],['close','Tutup galeri','×',close]].forEach(c=>{
var b=el('button','lightbox__btn lightbox__'+c[0],c[2]);
b.type='button';sa(b,'aria-label',c[1]);on(b,'click',c[3]);lb.append(b);
});
on(lb,'click',e=>{if(e.target===lb)close()});
var x0=null;
on(lb,'touchstart',e=>{x0=e.touches[0].clientX},{passive:true});
on(lb,'touchend',e=>{
if(x0===null)return;
var dx=e.changedTouches[0].clientX-x0;x0=null;
if(Math.abs(dx)>40)go(dx<0?1:-1);
});
B.append(lb);
}
function go(n){
I=(I+n+L.length)%L.length;
lbImg.src=L[I];lbN.textContent=lbImg.alt=I+1+' / '+L.length;
}
function close(){lb.hidden=true;cl(R,'is-lb',false);if(from)from.focus()}
$$('[data-gallery]').forEach(g=>{
var links=$$('a[href]',g);
links.forEach((a,i)=>on(a,'click',e=>{
e.preventDefault();
if(!lb)lbBuild();
L=links.map(x=>x.href);I=i;from=a;go(0);
cl(lb,'is-single',L.length<2);lb.hidden=false;cl(R,'is-lb',true);
$('.lightbox__close',lb).focus();
}));
});
on(d,'keydown',e=>{
if(!lb||lb.hidden)return;
var k=e.key;
if(k==='Escape')close();
else if(k==='ArrowRight')go(1);
else if(k==='ArrowLeft')go(-1);
});
// video YouTube: iframe dibuat saat diklik
$$('[data-yt]').forEach(box=>{
var src=at(box,'data-yt'),id=(src.match(/embed\/([\w-]{11})/)||[])[1];
if(id){
var im=el('img','video__thumb');
im.alt='';im.loading='lazy';im.src='https://i.ytimg.com/vi/'+id+'/hqdefault.jpg';
box.prepend(im);
}
on(box,'click',()=>{
var f=el('iframe','video__frame');
f.src=src+'?autoplay=1&rel=0';f.title='Video';f.allow='autoplay; encrypted-media; picture-in-picture; fullscreen';f.allowFullscreen=true;
box.textContent='';box.append(f);cl(box,'is-playing',true);
if(au&&!au.paused)au.pause();
},{once:true});
});
// dock aktif
var dock=$('[data-dock]');
if(dock&&IO){
var map=new Map(),dl=$$('a[href^="#"]',dock);
dl.forEach(a=>{var s=d.getElementById(at(a,'href').slice(1));if(s)map.set(s,a)});
var dio=new IntersectionObserver(es=>es.forEach(x=>{
if(x.isIntersecting)dl.forEach(l=>cl(l,'is-active',l===map.get(x.target)));
}),{rootMargin:'-45% 0px -50% 0px'});
map.forEach((a,s)=>dio.observe(s));
}
})();
