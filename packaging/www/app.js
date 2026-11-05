// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        packaging/www/app.js
// Ersteller:    Wolfram Henkel
// Datum:        05. Nov. 2026
// Version:      0.3.0
//
// Copyright (c) 2026 Wolfram Henkel.
//
// Lizenz:       CCU-ModbusClient-Lizenz für kostenfreie Nutzung und
//               Weitergabe, Version 1.1.
//
// Private und gewerbliche Nutzung sind erlaubt.
// Die Software und Änderungen dürfen nur kostenlos unter
// derselben Lizenz und mit vollständigem Quellcode weitergegeben werden.
//
// Vollständiger Lizenztext: LICENSE
// -----------------------------------------------------------------------------

'use strict';
const $=id=>document.getElementById(id),sid=new URLSearchParams(location.search).get('sid')||'';
let cfg={schema:2,devices:[]},di=-1,ri=-1,loaded=false,saving=false;
const editButtons=['save','addDevice','removeDevice','addReg','removeReg'];
for(const id of editButtons)$(id).disabled=true;
const api=action=>`api.cgi?sid=${encodeURIComponent(sid)}&action=${action}`;
const notify=msg=>$('message').textContent=msg;
function device(){return cfg.devices[di]||null}
function reg(){return device()?.registers?.[ri]||null}
function newID(prefix,used){const ids=new Set(used);let candidate;do{candidate=`${prefix}-${Date.now()}-${++newID.sequence}`}while(ids.has(candidate));return candidate}
newID.sequence=0;
function refresh(){
 $('device').replaceChildren();cfg.devices.forEach((d,i)=>$('device').add(new Option(d.name||d.id,String(i))));
 if(di>=cfg.devices.length)di=cfg.devices.length-1;
 $('device').value=String(di);
 const d=device();for(const [k,v] of Object.entries({name:d?.name||'',host:d?.host||'',port:d?.port??502,unitID:d?.unit_id??1,timeout:d?.timeout_ms??1500,retries:d?.retries??0}))$(k).value=v;
 $('enabled').checked=!!d?.enabled;
 $('register').replaceChildren();(d?.registers||[]).forEach((r,i)=>$('register').add(new Option(r.name||r.id,String(i))));
 if(ri>=(d?.registers?.length||0))ri=(d?.registers?.length||0)-1;$('register').value=String(ri);
 const r=reg();for(const [k,v] of Object.entries({regName:r?.name||'',regType:r?.type||'holding',regAddr:r?.address??0,regData:r?.datatype||'uint16',regDirection:r?.numeric_direction||'',factor:r?.factor??1,poll:r?.poll_seconds??1,coilDirection:r?.coil_direction||'',offset:r?.offset??0,regUnit:r?.unit||'',measurement:r?.measurement||'',roomID:r?.room_id??0,functionID:r?.function_id??0}))$(k).value=v;
 for(const [key,id] of Object.entries({enabled:'regEnabled',byte_swap:'byteSwap',word_swap:'wordSwap',dword_swap:'dwordSwap',true_means_open:'trueMeansOpen',invert_input:'invertInput'}))$(id).checked=!!r?.[key];
}
function pull(){const d=device();if(!d)return;for(const k of ['name','host'])d[k]=$(k).value.trim();d.port=Number($('port').value);d.unit_id=Number($('unitID').value);d.enabled=$('enabled').checked;d.timeout_ms=Number($('timeout').value);d.retries=Number($('retries').value);const r=reg();if(r){r.name=$('regName').value.trim();r.type=$('regType').value;r.address=Number($('regAddr').value);r.datatype=$('regData').value;r.numeric_direction=$('regDirection').value;r.factor=Number($('factor').value);r.poll_seconds=Number($('poll').value);r.coil_direction=$('coilDirection').value;r.offset=Number($('offset').value);r.unit=$('regUnit').value.trim();r.measurement=$('measurement').value;r.room_id=Number($('roomID').value);r.function_id=Number($('functionID').value);for(const [key,id] of Object.entries({enabled:'regEnabled',byte_swap:'byteSwap',word_swap:'wordSwap',dword_swap:'dwordSwap',true_means_open:'trueMeansOpen',invert_input:'invertInput'}))r[key]=$(id).checked}}
$('device').onchange=()=>{pull();di=Number($('device').value);ri=-1;refresh()};
$('register').onchange=()=>{pull();ri=Number($('register').value);refresh()};
$('addDevice').onclick=()=>{if(!loaded||saving)return;pull();const id=newID('dev',cfg.devices.map(d=>d.id));cfg.devices.push({id,name:'Neues Gerät',host:'192.168.0.1',port:502,unit_id:1,enabled:false,timeout_ms:1500,retries:0,registers:[]});di=cfg.devices.length-1;ri=-1;refresh()};
$('removeDevice').onclick=()=>{if(!loaded||saving)return;if(di<0||!confirm('Gerät entfernen?'))return;cfg.devices.splice(di,1);di=-1;ri=-1;refresh()};
$('addReg').onclick=()=>{if(!loaded||saving)return;pull();const d=device();if(!d)return;d.registers.push({id:newID('reg',d.registers.map(r=>r.id)),name:'Neues Register',enabled:false,type:'holding',address:0,datatype:'uint16',factor:1,offset:0,poll_seconds:1});ri=d.registers.length-1;refresh()};
$('removeReg').onclick=()=>{if(!loaded||saving)return;if(ri<0||!confirm('Register entfernen?'))return;pull();device().registers.splice(ri,1);ri=-1;refresh()};
$('regType').addEventListener('change',()=>{if(['coil','discrete'].includes($('regType').value)){$('regData').value='bool';$('regDirection').value='';}else if($('regData').value==='bool'){$('regData').value='uint16';}$('coilDirection').value='';});
$('save').onclick=async()=>{if(!loaded||saving)return;pull();saving=true;$('save').disabled=true;try{const response=await fetch(api('write'),{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(cfg)});if(!response.ok)throw new Error(`HTTP ${response.status}`);const data=await response.json();if(!data.ok)throw new Error(data.error||'Speichern fehlgeschlagen');notify('Gespeichert. Dienst-Neustart erforderlich.')}catch(e){notify(`Fehler: ${e.message}`)}finally{saving=false;$('save').disabled=!loaded}};
(async()=>{try{const res=await fetch(api('read'));if(!res.ok)throw new Error(`HTTP ${res.status}`);cfg=await res.json();if(!Array.isArray(cfg.devices))throw new Error('Ungültige Konfiguration');di=cfg.devices.length?0:-1;refresh();loaded=true;for(const id of editButtons)$(id).disabled=false;$('status').textContent='Konfiguration geladen';}catch(e){$('status').textContent=`Fehler beim Laden: ${e.message}`}})();
