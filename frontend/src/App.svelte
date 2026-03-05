<script> 
import './style.css'; 
import { ToggleProtections, UpdateHotkey, SetAutoStart, GetCurrentConfig, UpdateBannedWords, ReadLogs } from '../wailsjs/go/main/App'; 
import { EventsOn } from '../wailsjs/runtime/runtime'; 
import { onMount } from 'svelte';

let bannedWords = $state(""); 
let isRunning = $state(false); 
let logTokens = $state([]); 
let currentSentence = $state(""); 
let autoStart = $state(false); 

let hotkeyDisplay = $state("CTRL + SHIFT + DELETE"); 
let hotkeyCodes = $state([17, 16, 46]); 
let isEditingHotkey = $state(false); 

onMount(async () => {
    // Load current configuration and sync UI states [cite: 5]
    const config = await GetCurrentConfig();
    isRunning = config.isRunning;
    hotkeyCodes = config.hotkeyCodes;
    bannedWords = config.bannedWords || "";
    autoStart = config.autoStart || false;
    
    formatHotkeyText(hotkeyCodes);

    const oldLogs = await ReadLogs();
    if (oldLogs) {
        logTokens = [{ text: oldLogs, type: "normal" }];
    }
    
    setTimeout(() => {
        const el = document.getElementById('log-box');
        if (el) el.scrollTop = el.scrollHeight;
    }, 150);
});

function formatHotkeyText(codes) {
    const names = codes.map(c => {
        if(c === 17) return "CTRL";
        if(c === 16) return "SHIFT";
        if(c === 18) return "ALT";
        if(c === 46) return "DELETE";
        if(c === 13) return "ENTER";
        if(c >= 48 && c <= 90) return String.fromCharCode(c);
        return "KEY-" + c;
    });
    hotkeyDisplay = names.join(" + ");
}

EventsOn("status-updated", (status) => { 
    isRunning = status; 
});

EventsOn("new-key-event", (data) => { 
    const key = data.text; 
    const specialKeys = ["[BACKSPACE]", "[SHIFT]", "[CTRL]", "[TAB]", "[CAPSLOCK]", "[ENTER]"]; 
    
    if (specialKeys.includes(key)) { 
        if (currentSentence !== "") flushSentence(); 
        logTokens = [...logTokens, { text: key === "[ENTER]" ? "\n" : key, type: "function-key" }]; 
    } else if (key === " ") { 
        flushSentence(); 
        logTokens = [...logTokens, { text: " ", type: "normal" }]; 
    } else { 
        currentSentence += key; 
    } 
    
    setTimeout(() => { 
        const el = document.getElementById('log-box'); 
        if (el) el.scrollTop = el.scrollHeight; 
    }, 10); 
});

function flushSentence() { 
    if (currentSentence.trim() !== "") { 
        const list = bannedWords.split(/[, ]+/).map(w => w.trim().toLowerCase());
        const isBanned = list.some(word => word !== "" && currentSentence.toLowerCase().includes(word));
        logTokens = [...logTokens, { text: currentSentence, type: isBanned ? "banned" : "normal" }]; 
    } 
    currentSentence = "";
} 

function captureHotkey(e) { 
    if (!isEditingHotkey) return;
    if (e.keyCode === 13 && !e.ctrlKey && !e.shiftKey) return;
    
    e.preventDefault(); 
    let pressedKeys = []; 
    if (e.ctrlKey) pressedKeys.push({name: "CTRL", code: 17}); 
    if (e.shiftKey) pressedKeys.push({name: "SHIFT", code: 16});
    if (e.altKey) pressedKeys.push({name: "ALT", code: 18}); 
    
    const keyName = e.key.toUpperCase();
    if (!["CONTROL", "SHIFT", "ALT"].includes(keyName)) { 
        pressedKeys.push({name: keyName, code: e.keyCode});
    } 
    
    const finalKeys = pressedKeys.slice(0, 3); 
    hotkeyDisplay = finalKeys.map(k => k.name).join(" + ");
    hotkeyCodes = finalKeys.map(k => k.code); 
} 

async function handleAction() { 
    const newStatus = !isRunning;
    isEditingHotkey = false;
    await ToggleProtections(newStatus); 
} 

$effect(() => { 
    if (bannedWords !== undefined) {
        UpdateBannedWords(bannedWords);
    } 
});
</script>

<main class="h-screen bg-base-200 p-4 flex flex-col gap-3 font-sans overflow-hidden"> 
  <div class="flex justify-between items-center px-2 shrink-0"> 
    <div> 
      <h1 class="text-xl font-black text-[#0095A0] uppercase italic tracking-tighter">Protect Children v2.0</h1> 
      <p class="text-[8px] font-bold opacity-40 uppercase tracking-widest">by Rumix Tools</p> 
    </div> 
    <div class="flex items-center gap-2 bg-base-100 px-3 py-1 rounded-full border border-base-300 shadow-sm"> 
      <div class="badge badge-xs {isRunning ? 'bg-[#E43C2F] animate-pulse' : 'bg-[#0095A0]'} border-none"></div> 
      <span class="text-[11px] font-black opacity-60 uppercase">{isRunning ? 'Monitoring Active' : 'System Ready'}</span> 
    </div> 
  </div> 

  <div class="flex gap-4 flex-1 overflow-hidden"> 
    <div class="flex-1 flex flex-col h-full overflow-hidden"> 
      <div class="bg-base-100 rounded-xl border border-base-300 shadow-lg overflow-hidden flex flex-col flex-1"> 
        <div class="bg-base-200/50 h-10 px-4 border-b border-base-300 flex justify-between items-center shrink-0"> 
          <h2 class="text-[10px] font-black uppercase tracking-widest opacity-60">Typing Activity</h2> 
        </div> 
        <div id="log-box" class="flex-1 overflow-y-auto p-4 font-mono text-[14px] leading-relaxed bg-white no-scrollbar whitespace-pre-wrap"> 
          {#each logTokens as token} 
            {#if token.type === 'banned'} 
              <span class="bg-[#E43C2F] text-white px-1.5 py-0.5 rounded font-bold shadow-sm inline-block my-0.5">{token.text}</span> 
            {:else if token.type === 'function-key'} 
              <span class="bg-blue-900 text-white px-1.5 py-0.5 rounded font-bold text-[11px] mx-0.5 shadow-sm inline-block my-0.5 uppercase">{token.text}</span> 
            {:else} 
              <span class="text-gray-800">{token.text}</span> 
            {/if} 
          {/each} 
          <span class="text-blue-500 italic font-bold">{currentSentence}</span> 
        </div> 
      </div> 
      <div class="mt-2 flex justify-between items-center bg-blue-900 text-white p-2 px-4 rounded-lg shadow-md shrink-0"> 
        <span class="text-[11px] font-medium uppercase tracking-widest opacity-60">Recovery Hotkey:</span> 
        <code class="text-[11px] font-medium tracking-widest">{hotkeyDisplay}</code> 
      </div> 
    </div> 

    <div class="w-[260px] flex flex-col h-full overflow-hidden"> 
      <div class="bg-base-100 rounded-xl border border-base-300 shadow-lg overflow-hidden flex flex-col flex-1"> 
        <div class="bg-base-200/50 h-10 px-4 border-b border-base-300 flex items-center shrink-0"> 
          <h2 class="text-[10px] font-black uppercase tracking-widest opacity-60">CONTROL PANEL</h2> 
        </div> 
        <div class="p-4 flex flex-col gap-4 overflow-y-auto no-scrollbar"> 
          <div class="flex flex-col gap-3"> 
            <div class="flex items-center justify-between"> 
              <span class="text-[12px] font-bold">Autostart (On Boot)</span> 
              <input type="checkbox" class="toggle toggle-sm border-[#0095A0] checked:bg-[#0095A0] checked:border-[#0095A0]" style="--tglbg: #FFFFFF !important;" bind:checked={autoStart} disabled={isRunning} onchange={async () => { await SetAutoStart(autoStart); }} /> 
            </div> 
            <div class="flex items-center justify-between"> 
              <span class="text-[12px] font-bold">Hotkey</span> 
              <button class="btn btn-xs {isEditingHotkey ? 'btn-primary' : 'btn-outline border-base-300'}" disabled={isRunning} onclick={async () => { if(isEditingHotkey) { await UpdateHotkey(hotkeyCodes); isEditingHotkey = false; } else { isEditingHotkey = true; } }}> {isEditingHotkey ? 'Save' : 'Change'} </button> 
            </div> 
            {#if isEditingHotkey} 
              <input type="text" readonly value={hotkeyDisplay} onkeydown={captureHotkey} class="input input-bordered input-xs text-center font-bold text-blue-900 animate-pulse bg-blue-50 w-full" /> 
            {/if} 
          </div> 
          <div class="form-control"> 
            <label class="label p-0 mb-1"> 
              <span class="label-text font-medium text-[12px] text-[#E43C2F] tracking-tighter">Sensor List (Comma)</span> 
            </label> 
            <textarea bind:value={bannedWords} disabled={isRunning} class="textarea textarea-bordered h-28 text-[12px] leading-tight resize-none focus:outline-none focus:border-[#0095A0]" placeholder="Example: gambling, casino..."></textarea> 
          </div> 
          <button class="btn btn-md w-full transition-all active:scale-95 {isRunning ? 'bg-[#E43C2F] text-white border-none' : 'bg-[#0095A0] text-white border-none'} text-xs" onclick={handleAction}> {isRunning ? 'STOP MONITORING' : 'START MONITORING'} </button> 
        </div> 
      </div> 
    </div> 
  </div> 
</main>