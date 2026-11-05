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

(function(){
'use strict';
var sid=new URLSearchParams(location.search).get('sid')||'';
var cfg=null, states=[], dirty=false, editDevice=-1, editRegister=-1, expanded={};
var nextSerial=0;
function generateID(prefix){var ids=new Set();if(cfg)cfg.devices.forEach(function(d){ids.add(d.id);(d.registers||[]).forEach(function(r){ids.add(r.id);});});var x;do{x=prefix+'-'+Date.now()+'-'+(++nextSerial);}while(ids.has(x));return x;}
var $=function(id){return document.getElementById(id);};
$('saveAll').disabled=true;
$('addDevice').disabled=true;

function api(action,method,body){
  var o={method:method||'GET',cache:'no-store'};
  action=({config:'read',save:'write'})[action]||action;
  if(body!==undefined){o.headers={'Content-Type':'application/json'};o.body=JSON.stringify(body);}
  return fetch('api.cgi?sid='+encodeURIComponent(sid)+'&action='+action,o).then(function(r){
    return r.text().then(function(t){
      var j; try{j=JSON.parse(t);}catch(e){throw new Error('Ungueltige Antwort: '+t.slice(0,100));}
      if(!r.ok || j.ok===false) throw new Error(j.error||('HTTP '+r.status));
      return j;
    });
  });
}
function esc(s){return String(s==null?'':s).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c];});}
function setDirty(v){dirty=v;$('dirtyText').textContent=v?'Ungespeicherte Änderungen.':'Keine ungespeicherten Änderungen.';}
function notice(msg,err){var n=$('notice');n.textContent=msg;n.className='notice'+(err?' error':'');setTimeout(function(){n.className='notice hidden';},6000);}
function stateText(d){return d.enabled?'Aktiviert':'Deaktiviert';}

function render(){
  var root=$('devices');root.innerHTML='';
  if(!cfg||!cfg.devices||!cfg.devices.length){root.innerHTML='<div class="empty">Noch keine Modbus-Geräte angelegt.</div>';return;}
  cfg.devices.forEach(function(d,idx){
    var c=document.createElement('div');c.className='card';
    c.innerHTML='<div class="cardhead"><span class="lamp '+(d.enabled?'green':'gray')+'"></span><div><h3>'+esc(d.name)+'</h3><div class="meta">'+esc(d.host)+' : '+d.port+' · Unit '+d.unit_id+'</div><div class="state">'+esc(stateText(d))+' · '+(d.registers||[]).length+' Register'+'</div></div></div>'+
      '<div class="actions"><button data-values="'+idx+'">Register anzeigen</button><button data-edit="'+idx+'">Bearbeiten</button></div>'+
      '<div class="values hidden" id="values-'+idx+'"></div>';
    root.appendChild(c);
    if(expanded[d.id]){
      var box=$('values-'+idx);
      if(box){
        box.classList.remove('hidden');
        renderValues(idx, box);
      }
    }
  });
  root.querySelectorAll('[data-edit]').forEach(function(b){b.onclick=function(){openDevice(Number(this.getAttribute('data-edit')));};});
  root.querySelectorAll('[data-values]').forEach(function(b){b.onclick=function(){toggleValues(Number(this.getAttribute('data-values')));};});
}
function renderValues(idx,box){
  var d=cfg.devices[idx];
  box.innerHTML='';
  if(!d.registers.length){box.textContent='Keine Register vorhanden.';return;}
  d.registers.forEach(function(r){
    var line=document.createElement('div');line.className='valueRow';
    var title=document.createElement('span');title.textContent=r.name||r.id;
    var info=document.createElement('span');info.textContent=typeName(r.type)+' · '+r.datatype+' · Adresse '+r.address+(r.unit?' · '+r.unit:'');
    line.appendChild(title);line.appendChild(info);box.appendChild(line);
  });
}

function toggleValues(idx){
  var box=$('values-'+idx), d=cfg.devices[idx];
  box.classList.toggle('hidden');
  expanded[d.id]=!box.classList.contains('hidden');
  if(expanded[d.id])renderValues(idx, box);
}

function blankDevice(){return {id:generateID('dev'),name:'',enabled:true,host:'',port:502,unit_id:1,timeout_ms:1500,retries:0,registers:[]};}
function openDevice(idx){
  editDevice=idx;var d=idx<0?blankDevice():JSON.parse(JSON.stringify(cfg.devices[idx]));if(!Array.isArray(d.registers))d.registers=[];
  $('deviceDialog')._working=d;$('deviceTitle').textContent=idx<0?'Gerät hinzufügen':'Gerät bearbeiten';
  $('dName').value=d.name;$('dHost').value=d.host;$('dPort').value=d.port;$('dUnit').value=d.unit_id;$('dTimeout').value=d.timeout_ms;$('dRetries').value=d.retries;$('dEnabled').checked=d.enabled;['dHost','dPort','dUnit'].forEach(function(id){$(id).disabled=idx>=0 && !!d.ccu_address;});
  $('connectionResult').textContent='';$('deleteDevice').style.visibility=idx<0?'hidden':'visible';renderRegisters();$('deviceDialog').showModal();
}
function workingDevice(){
  var d=$('deviceDialog')._working;
  d.name=$('dName').value.trim();d.host=$('dHost').value.trim();d.port=Number($('dPort').value);d.unit_id=Number($('dUnit').value);d.timeout_ms=Number($('dTimeout').value);d.retries=Number($('dRetries').value);d.enabled=$('dEnabled').checked;return d;
}
function renderRegisters(){
  var d=$('deviceDialog')._working, tb=$('registerRows');tb.innerHTML='';
  d.registers.forEach(function(r,i){var tr=document.createElement('tr');tr.innerHTML='<td><span class="regName">'+esc(r.name)+'</span></td><td>'+esc(typeName(r.type)+(r.type==='coil'?' ('+(r.coil_direction==='input'?'Eingang':'Ausgang')+')':''))+'</td><td>'+r.address+'</td><td>'+esc(r.datatype)+'</td><td>'+r.poll_seconds+' s</td><td><button type="button" data-r="'+i+'">Bearbeiten</button></td>';tb.appendChild(tr);});
  tb.querySelectorAll('[data-r]').forEach(function(b){b.onclick=function(){openRegister(Number(this.getAttribute('data-r')));};});
}
function typeName(t){return {holding:'Holding',input:'Input',coil:'Coil',discrete:'Discrete'}[t]||t;}
function blankRegister(){return {id:generateID('reg'),name:'',enabled:true,type:'holding',coil_direction:'',numeric_direction:'',address:0,datatype:'uint16',factor:1,offset:0,byte_swap:false,word_swap:false,dword_swap:false,poll_seconds:10,unit:'',measurement:''};}
function openRegister(idx){
  editRegister=idx;var d=$('deviceDialog')._working,r=idx<0?blankRegister():JSON.parse(JSON.stringify(d.registers[idx]));$('registerDialog')._working=r;$('registerTitle').textContent=idx<0?'Register hinzufügen':'Register bearbeiten';
  $('rName').value=r.name;$('rType').value=r.type;$('rCoilDirection').value=r.coil_direction||'output';$('rAddress').value=r.address;$('rDatatype').value=String(r.datatype).toUpperCase();$('rNumericDirection').value=r.numeric_direction||'';$('rMeasurement').value=r.measurement||'';$('rUnit').value=r.unit||'';$('rFactor').value=r.factor;$('rOffset').value=r.offset;$('rPoll').value=r.poll_seconds;$('rByteSwap').checked=r.byte_swap;$('rWordSwap').checked=r.word_swap;$('rDWordSwap').checked=!!r.dword_swap;$('rInvertInput').checked=!!r.invert_input;$('rTrueMeansOpen').checked=!!r.true_means_open;$('rEnabled').checked=r.enabled;
  $('registerResult').textContent='';$('deleteRegister').style.visibility=idx<0?'hidden':'visible';
  $('rSeries').checked=false;$('rSeries').disabled=idx>=0;$('rSeriesCount').value=2;syncSeries();syncDatatype();$('registerDialog').showModal();
}
function workingRegister(){
  var r=$('registerDialog')._working;r.name=$('rName').value.trim();r.type=$('rType').value;r.coil_direction=r.type==='coil'?$('rCoilDirection').value:'';r.address=Number($('rAddress').value);r.datatype=$('rDatatype').value.toLowerCase();r.numeric_direction=r.type==='holding'?$('rNumericDirection').value:'';r.measurement=['holding','input'].includes(r.type)?$('rMeasurement').value:'';r.unit=$('rUnit').value.trim();r.factor=parseNumber($('rFactor').value,1);r.offset=parseNumber($('rOffset').value,0);r.poll_seconds=Number($('rPoll').value);r.byte_swap=$('rByteSwap').checked;r.word_swap=$('rWordSwap').checked;r.dword_swap=$('rDWordSwap').checked;r.invert_input=$('rInvertInput').checked;r.true_means_open=$('rTrueMeansOpen').checked;r.enabled=$('rEnabled').checked;return r;
}
function parseNumber(v,def){var text=String(v).trim();if(!text)return def;return Number(text.replace(',','.'));}
function numberedName(name,n){
  var m=String(name||'').match(/^(.*?)(\d+)\s*$/);
  if(m)return m[1]+n;
  return String(name||'').replace(/\s+$/,'')+' '+n;
}
function registerStride(r){
  if(r.type==='coil'||r.type==='discrete')return 1;
  if(r.datatype==='int64'||r.datatype==='uint64')return 4;return (r.datatype==='int32'||r.datatype==='uint32'||r.datatype==='float32')?2:1;
}
function syncSeries(){
  var on=$('rSeries').checked;
  $('rSeriesCountWrap').style.display=on?'':'none';
}
function syncDatatype(){
  var t=$('rType').value;
  $('rCoilDirectionWrap').style.display=t==='coil'?'':'none';
  $('rInvertInputWrap').style.display=(t==='discrete'||(t==='coil'&&$('rCoilDirection').value==='input'))?'':'none';
  if(t==='coil'||t==='discrete'){
    $('rDatatype').value='BOOL';$('rDatatype').disabled=true;
  }else{
    $('rDatatype').disabled=false;
    if($('rDatatype').value==='BOOL')$('rDatatype').value='UINT16';
  }
  var dt=$('rDatatype').value.toLowerCase();
  $('precisionHint').hidden=!(dt==='int64'||dt==='uint64');
  $('rNumericDirection').disabled=t!=='holding'||dt==='int64'||dt==='uint64'||dt==='bool';
  if($('rNumericDirection').disabled)$('rNumericDirection').value='';
  $('rMeasurement').disabled=(t!=='holding'&&t!=='input')||dt==='int64'||dt==='uint64'||dt==='bool';
  if($('rMeasurement').disabled)$('rMeasurement').value='';
}
function validRegister(r){
  if(!r.name)return 'Bitte eine Bezeichnung eingeben.';
  if(!Number.isInteger(r.address)||r.address<0||r.address>65535)return 'Registeradresse muss zwischen 0 und 65535 liegen.';
  if(!Number.isInteger(r.poll_seconds)||r.poll_seconds<1||r.poll_seconds>86400)return 'Intervall muss zwischen 1 und 86400 s liegen.';
  if(!Number.isFinite(r.factor)||!Number.isFinite(r.offset))return 'Faktor und Offset müssen endliche Zahlen sein.';
  if(r.address+registerStride(r)-1>65535)return 'Die Registeradresse überschreitet mit dem Datentyp 65535.';
  if((r.datatype==='int64'||r.datatype==='uint64')&&r.numeric_direction==='output')return '64-Bit-Ausgänge sind nicht unterstützt.';
  if((r.datatype==='int64'||r.datatype==='uint64')&&r.measurement&&r.measurement!=='generic')return '64-Bit-Messgrößen sind noch nicht unterstützt.';
  if((r.type==='coil'||r.type==='discrete')&&r.datatype!=='bool')return 'Coils/Discrete müssen BOOL sein.';
  return '';
}
function validDevice(d){
  if(!d.name||!d.host)return 'Name und IP-Adresse/Host sind Pflichtfelder.';
  if(!Number.isInteger(d.port)||d.port<1||d.port>65535)return 'Port muss zwischen 1 und 65535 liegen.';
  if(!Number.isInteger(d.unit_id)||d.unit_id<0||d.unit_id>255)return 'Unit-ID muss zwischen 0 und 255 liegen.';
  if(!Number.isInteger(d.timeout_ms)||d.timeout_ms<100||d.timeout_ms>30000)return 'Timeout muss zwischen 100 und 30000 ms liegen.';
  if(!Number.isInteger(d.retries)||d.retries<0||d.retries>2)return 'Wiederholungen müssen zwischen 0 und 2 liegen.';
  for(var r of d.registers){var e=validRegister(r);if(e)return r.name+': '+e;}
  return '';
}


$('addDevice').onclick=function(){openDevice(-1);};
$('addRegister').onclick=function(){openRegister(-1);};
$('rType').onchange=syncDatatype;
$('rDatatype').onchange=syncDatatype;
$('rNumericDirection').onchange=syncDatatype;
$('rCoilDirection').onchange=syncDatatype;
$('rSeries').onchange=syncSeries;
$('applyRegister').onclick=function(){
  var r=workingRegister();
  var validation=validRegister(r);if(validation){notice(validation,true);return;}
  var d=$('deviceDialog')._working;
  if(editRegister>=0){
    d.registers[editRegister]=r;
  }else if($('rSeries').checked){
    var count=Number($('rSeriesCount').value);
    if(!Number.isInteger(count)||count<2||count>256){notice('Die Gesamtanzahl muss zwischen 2 und 256 liegen.',true);return;}
    var stride=registerStride(r), start=r.address;
    if(start+count*stride-1>65535){notice('Die Registerserie überschreitet Adresse 65535.',true);return;}
    for(var n=1;n<=count;n++){
      var x=JSON.parse(JSON.stringify(r));
      x.id=generateID('reg');
      x.name=numberedName(r.name,n);
      x.address=start+(n-1)*stride;
      d.registers.push(x);
    }
  }else{
    d.registers.push(r);
  }
  $('registerDialog').close();renderRegisters();
};
$('deleteRegister').onclick=function(){if(editRegister>=0&&confirm('Dieses Register wirklich löschen?')){$('deviceDialog')._working.registers.splice(editRegister,1);$('registerDialog').close();renderRegisters();}};
$('applyDevice').onclick=function(){var d=workingDevice();var e=validDevice(d);if(e){notice(e,true);return;}if(editDevice<0)cfg.devices.push(d);else cfg.devices[editDevice]=d;$('deviceDialog').close();setDirty(true);render();};
$('deleteDevice').onclick=function(){if(editDevice>=0&&confirm('Gerät und alle zugehörigen Register wirklich löschen?')){cfg.devices.splice(editDevice,1);$('deviceDialog').close();setDirty(true);render();}};
$('testConnection').onclick=function(){var out=$('connectionResult'),d=workingDevice();if(!d.name||!d.host){out.textContent='Name und Host fehlen.';return;}out.className='';out.textContent='Prüfe…';api('test_connection','POST',d).then(function(x){out.className='ok';out.textContent='✓ TCP-Verbindung erfolgreich';}).catch(function(e){out.className='bad';out.textContent='✗ '+e.message;});};
$('testRegister').onclick=function(){var out=$('registerResult'),d=workingDevice(),r=workingRegister();var e=validRegister(r);if(e){out.textContent=e;return;}if(!d.name||!d.host){out.textContent='Gerätename und Host fehlen.';return;}out.className='';out.textContent='Lese…';api('test_register','POST',{device:d,register:r}).then(function(x){out.className='ok';out.textContent='✓ Rohwert: '+x.raw+' · Wert: '+x.value+(x.unit?' '+x.unit:'');}).catch(function(e){out.className='bad';out.textContent='✗ '+e.message;});};
$('saveAll').onclick=function(){
  if(!dirty)return;
  this.disabled=true;var btn=this;
  api('save','POST',cfg).then(function(){
    return api('config').then(function(fresh){if(!fresh||!Array.isArray(fresh.devices))throw new Error('Gespeichert, aber nicht neu ladbar');cfg=fresh;cfg.devices.forEach(function(d){if(!Array.isArray(d.registers))d.registers=[];});setDirty(false);expanded={};render();notice('Konfiguration gespeichert. Der Client-Dienst muss neu gestartet werden.');});
  }).catch(function(e){notice('Speichern fehlgeschlagen: '+e.message,true);})
  .finally(function(){btn.disabled=false;});
};

function init(){
 api('config').then(function(data){
  if(!data||!Array.isArray(data.devices))throw new Error('Ungültige Konfiguration');
  cfg=data;
  cfg.devices.forEach(function(d){if(!Array.isArray(d.registers))d.registers=[];});
  setDirty(false);render();
  $('saveAll').disabled=false;$('addDevice').disabled=false;
 }).catch(function(e){notice('Konfiguration nicht geladen: '+e.message,true);$('saveAll').disabled=true;$('addDevice').disabled=true;});
}
window.addEventListener('beforeunload',function(e){if(dirty){e.preventDefault();e.returnValue='';}});
init();
})();
