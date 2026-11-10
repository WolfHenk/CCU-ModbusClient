// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        tools/ui_test.js
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

// Browser-independent UI workflow regression checks. No CCU access or writes.
const vm = require('node:vm');
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');
const root = path.resolve(__dirname, '..');
const script = fs.readFileSync(path.join(root, 'packaging/www/app.js'), 'utf8');
const html = fs.readFileSync(path.join(root, 'packaging/www/index.cgi'), 'utf8');
const ids = [...html.matchAll(/\bid="([^"]+)"/g)].map(x => x[1]);
assert.equal(new Set(ids).size, ids.length, 'every element id must be unique');
assert.ok(html.includes('<img src="img/modbus-logo.png" class="logo"'), 'original Modbus logo missing in header');
assert.ok(fs.statSync(path.join(root, 'packaging/www/img/modbus-logo.png')).size > 1000, 'Modbus logo file missing');

function clone(x){return JSON.parse(JSON.stringify(x))}
function makeRegister(name, datatype='int16', id='r1'){
 return {id,name,enabled:true,type:'input',address:10,datatype,factor:1,offset:0,poll_seconds:10,
         unit:'°C',byte_swap:false,word_swap:false,dword_swap:false,numeric_direction:'',measurement:''};
}
function makeDevice(registers=[makeRegister('Temperatur')]){
 return {id:'d1',name:'Erstes Gerät',enabled:true,host:'192.0.2.15',port:502,unit_id:1,
         ccu_address:'MBC020F01F601',timeout_ms:1500,retries:0,registers};
}
class Elem{
 constructor(doc,id=''){
  this.doc=doc;this.id=id;this.value='';this.checked=false;this.disabled=false;
  this.textContent='';this._html='';this.children=[];this.queries=new Map();
  this.attributes={};this.dataset={};this.className='';this.style={};this.hidden=false;this.modal=false;
  this.classList={
   contains:(c)=>this.className.split(/\s+/).includes(c),
   add:(c)=>{if(!this.classList.contains(c))this.className+=(this.className?' ':'')+c},
   remove:(c)=>{this.className=this.className.split(/\s+/).filter(x=>x!==c).join(' ')},
   toggle:(c)=>{if(this.classList.contains(c)){this.classList.remove(c);return false;}this.classList.add(c);return true}
  };
 }
 set innerHTML(markup){this._html=markup;this.children=[];this.queries.clear()}
 get innerHTML(){return this._html}
 appendChild(x){
  this.children.push(x);
  let all=x.innerHTML+' '+(x.id?(' id="'+x.id+'"'):'');
  for (const m of all.matchAll(/\bid="([^"]+)"/g)) {
   let e=new Elem(this.doc,m[1]);
   if(/class="[^"]*hidden/.test(all))e.className='values hidden';
   this.doc.elements[m[1]]=e;
  }
  return x;
 }
 querySelectorAll(sel){
  if(this.queries.has(sel))return this.queries.get(sel);
  const attr=sel.replace(/[\[\]]/g,'');
  const markup=[this.innerHTML,...this.children.map(x=>x.innerHTML)].join(' ');
  const matches=[...markup.matchAll(new RegExp(attr+'="([^"]+)"','g'))].map(m=>{
   const b=new Elem(this.doc);
   b.attributes[attr]=m[1];
   b.getAttribute=k=>b.attributes[k];
   return b;
  });
  this.queries.set(sel,matches);
  return matches;
 }
 showModal(){this.modal=true}
 close(){this.modal=false}
}
function setup(input, opts={}){
 const elements={};
 const doc={elements,getElementById(id){return elements[id]||null},createElement(){return new Elem(doc)}};
 for(const id of ids)elements[id]=new Elem(doc,id);
 let server=clone(input),writes=[],reads=[],requests=[],updateRequests=[];
 const mockfetch=async(url,init={})=>{
  const act=new URL('http://ccu.invalid/'+url).searchParams.get('action');
  if(url.startsWith('update-check.cgi')){
    updateRequests.push({act,method:init.method||'GET'});
    let result;
    if(act==='check')result={ok:true,current:'0.3.4',latest:opts.latest||'0.3.4',available:!!opts.latest};
    else if(act==='install')result={ok:true,started:true};
    else if(act==='status')result={ok:true,state:opts.updateState||'downloading',target:opts.latest||'0.3.5'};
    else throw new Error('unexpected update action '+act);
    return {ok:true,status:200,text:async()=>JSON.stringify(result)};
  }
  const body=init.body?JSON.parse(init.body):undefined;
  requests.push({act,body});
  let result={};
  if(act==='read'){reads.push(clone(server));result=clone(server)}
  else if(act==='write'){
   writes.push(clone(body));
   if(opts.rejectWrite) return {ok:false,status:400,text:async()=>JSON.stringify({ok:false,error:'Abgelehnt'})};
   server=clone(body);
   result={ok:true};
  } else if(act==='test_connection')result={ok:true};
  else if(act==='test_register')result={ok:true,raw:'18446744073709551615',value:'18446744073709551615'};
  else throw new Error('unexpected CGI action '+act);
  return {ok:true,status:200,text:async()=>JSON.stringify(result)};
 };
 const win={addEventListener(){},opener:null};
 const context={document:doc,window:win,location:{search:'?sid=test'},URLSearchParams,
  fetch:mockfetch,confirm:()=>true,Date,Set,Number,Math,JSON,Promise,
  encodeURIComponent,console,setTimeout:()=>{},setInterval:()=>{}};
 vm.createContext(context);
 vm.runInContext(script,context,{filename:'app.js'});
 const settle=()=>new Promise(ok=>setTimeout(ok,0));
 return {elements,requests,updateRequests,writes,reads,settle,context,readState:()=>vm.runInContext('cfg',context),
         getDirty:()=>elements.dirtyText.textContent==='Ungespeicherte Änderungen.'};
}
function click(root,selector,index=0){
 const arr=root.querySelectorAll(selector);
 assert.ok(arr[index],selector+' index '+index);
 assert.equal(typeof arr[index].onclick,'function',selector+' handler');
 return arr[index].onclick();
}
async function test(name,fn){
 try{await fn();console.log('PASS',name)}
 catch(e){console.error('FAIL',name,e.stack||e);process.exitCode=1}
}
(async()=>{
 await test('initialization and two device cards',async()=>{
  const env=setup({schema:2,devices:[makeDevice(),{...makeDevice([]),id:'d2',name:'Gerät 2',ccu_address:'MBC021001F601'}]});
  await env.settle();
  assert.equal(env.elements.devices.children.length,2);
  assert.equal(env.elements.dirtyText.textContent,'Keine ungespeicherten Änderungen.');
  assert.equal(env.elements.addDevice.disabled,false);
  assert.equal(env.writes.length,0);
 });
 await test('existing device preserves immutable CCU address and allows name edit',async()=>{
  const env=setup({schema:2,devices:[makeDevice()]});await env.settle();
  click(env.elements.devices,'[data-edit]');
  assert.equal(env.elements.dHost.disabled,true);
  assert.equal(env.elements.dUnit.disabled,true);
  env.elements.dName.value='Geändert';
  env.elements.applyDevice.onclick();
  assert.equal(env.getDirty(),true);
  assert.equal(env.elements.deviceDialog._working.ccu_address,'MBC020F01F601');
  await env.elements.saveAll.onclick();await env.settle();
  assert.equal(env.writes.length,1);
  assert.equal(env.writes[0].devices[0].name,'Geändert');
  assert.equal(env.writes[0].devices[0].ccu_address,'MBC020F01F601');
  assert.equal(env.getDirty(),false);
 });
 await test('existing 64-bit field retains exact configuration',async()=>{
  const r=makeRegister('Zähler','uint64');
  r.address=100;r.byte_swap=true;r.word_swap=true;r.dword_swap=true;r.measurement='generic';
  const env=setup({schema:2,devices:[makeDevice([r])]});await env.settle();
  click(env.elements.devices,'[data-edit]');
  click(env.elements.registerRows,'[data-r]');
  assert.equal(env.elements.rDatatype.value,'UINT64');
  assert.equal(env.elements.rDWordSwap.checked,true);
  assert.equal(env.elements.rByteSwap.checked,true);
  assert.equal(env.elements.rWordSwap.checked,true);
  assert.equal(env.elements.precisionHint.hidden,false);
  env.elements.applyRegister.onclick();env.elements.applyDevice.onclick();
  await env.elements.saveAll.onclick();await env.settle();
  const saved=env.writes[0].devices[0].registers[0];
  assert.equal(saved.datatype,'uint64');
  assert.equal(saved.address,100);
  assert.equal(saved.dword_swap,true);
  assert.equal(saved.byte_swap,true);
  assert.equal(saved.word_swap,true);
  assert.equal(saved.unit,'°C');
 });
 await test('64-bit register series respects four-word stride and unique IDs',async()=>{
  const env=setup({schema:2,devices:[makeDevice([])]});await env.settle();
  click(env.elements.devices,'[data-edit]');
  env.elements.addRegister.onclick();
  env.elements.rName.value='Zähler';
  env.elements.rType.value='input';env.elements.rType.onchange();
  env.elements.rDatatype.value='UINT64';env.elements.rDatatype.onchange();
  env.elements.rAddress.value='100';
  env.elements.rSeries.checked=true;env.elements.rSeries.onchange();
  env.elements.rSeriesCount.value='3';
  env.elements.applyRegister.onclick();
  const regs=env.elements.deviceDialog._working.registers;
  assert.equal(regs.length,3);
  assert.deepEqual(regs.map(r=>r.address),[100,104,108]);
  assert.equal(new Set(regs.map(r=>r.id)).size,3);
  assert.equal(regs.every(r=>r.datatype==='uint64'),true);
 });
 await test('reject 64-bit addresses exceeding 65535',async()=>{
  const env=setup({schema:2,devices:[makeDevice([])]});await env.settle();
  click(env.elements.devices,'[data-edit]');
  env.elements.addRegister.onclick();
  env.elements.rName.value='Überlauf';
  env.elements.rType.value='input';env.elements.rType.onchange();
  env.elements.rDatatype.value='UINT64';env.elements.rDatatype.onchange();
  env.elements.rAddress.value='65534';
  env.elements.applyRegister.onclick();
  assert.equal(env.elements.deviceDialog._working.registers.length,0);
  assert.equal(env.elements.registerDialog.modal,true);
 });
 await test('register and connection tests are nonpersistent and read-only',async()=>{
  const env=setup({schema:2,devices:[makeDevice()]});await env.settle();
  click(env.elements.devices,'[data-edit]');click(env.elements.registerRows,'[data-r]');
  await env.elements.testRegister.onclick();await env.settle();
  assert.match(env.elements.registerResult.textContent,/18446744073709551615/);
  env.elements.registerDialog.close();
  await env.elements.testConnection.onclick();await env.settle();
  assert.match(env.elements.connectionResult.textContent,/erfolgreich/);
  assert.equal(env.writes.length,0);
  assert.deepEqual(env.requests.map(x=>x.act),['read','test_register','test_connection']);
 });
 await test('failed write retains pending edits',async()=>{
  const env=setup({schema:2,devices:[makeDevice()]},{rejectWrite:true});await env.settle();
  click(env.elements.devices,'[data-edit]');env.elements.dName.value='Entwurf';env.elements.applyDevice.onclick();
  await env.elements.saveAll.onclick();await env.settle();
  assert.equal(env.getDirty(),true);
  assert.equal(env.writes.length,1);
 });
 await test('new device ID distinct, null registers load as empty',async()=>{
  const env=setup({schema:2,devices:[makeDevice(null)]});await env.settle();
  click(env.elements.devices,'[data-edit]');
  assert.equal(env.elements.deviceDialog._working.registers.length,0);
  env.elements.deviceDialog.close();
  env.elements.addDevice.onclick();
  assert.notEqual(env.elements.deviceDialog._working.id,'d1');
 });
 await test('update notice and button appear only for newer public version',async()=>{
  const up=setup({schema:2,devices:[]},{latest:'0.3.5'});
  await up.settle();
  assert.match(up.elements.updateInfo.textContent,/0.3.5 verfügbar/);
  assert.equal(up.elements.installUpdate.classList.contains('hidden'),false);
  assert.equal(up.elements.installUpdate.disabled,false);
  assert.deepEqual(up.updateRequests.map(x=>x.act),['check']);
  await up.elements.installUpdate.onclick();await up.settle();
  assert.ok(up.updateRequests.some(x=>x.act==='install'&&x.method==='POST'));
  assert.ok(up.updateRequests.some(x=>x.act==='status'));
  const current=setup({schema:2,devices:[]});await current.settle();
  assert.equal(current.elements.installUpdate.classList.contains('hidden'),true);
  assert.match(current.elements.updateInfo.textContent,/Aktuelle Version/);
 });
 await test('unsaved changes block updating',async()=>{
  const env=setup({schema:2,devices:[makeDevice()]},{latest:'0.3.5'});await env.settle();
  env.elements.addDevice.onclick();env.elements.dName.value='Test';
  env.elements.dHost.value='192.0.2.5';env.elements.applyDevice.onclick();
  assert.equal(env.getDirty(),true);
  await env.elements.installUpdate.onclick();await env.settle();
  assert.equal(env.updateRequests.some(x=>x.act==='install'),false);
 });
 if(!process.exitCode)console.log('UI_WORKFLOW_TESTS_PASS');
})();
