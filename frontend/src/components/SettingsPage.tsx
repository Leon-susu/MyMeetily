import React, { useEffect, useState } from 'react';
import { AppService } from '../hooks/useWailsEvents';
import type { ModelOption, Preferences } from '../types';

const fieldStyle: React.CSSProperties = { width: '100%', boxSizing: 'border-box', border: '1px solid #cbd5e1', borderRadius: 8, padding: '10px 11px', background: '#fff', color: '#334155' };
const labelStyle: React.CSSProperties = { display: 'block', color: '#475569', fontSize: 13, fontWeight: 650, marginBottom: 6 };

export const SettingsPage: React.FC = () => {
  const [form, setForm] = useState<Preferences | null>(null);
  const [whisperModels, setWhisperModels] = useState<ModelOption[]>([]);
  const [ollamaModels, setOllamaModels] = useState<ModelOption[]>([]);
  const [message, setMessage] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    void Promise.all([
      AppService().GetPreferences(),
      AppService().ListWhisperModels(),
      AppService().ListOllamaModels().catch(() => []),
    ]).then(([preferences, whispers, ollamas]) => {
      setForm(preferences); setWhisperModels(whispers || []); setOllamaModels(ollamas || []);
    }).catch(error => setMessage(`載入設定失敗：${String(error)}`));
  }, []);

  if (!form) return <div style={{ padding: 32, color: '#64748b' }}>正在載入設定…</div>;
  const update = <K extends keyof Preferences>(key: K, value: Preferences[K]) => setForm({ ...form, [key]: value });
  const save = async () => {
    setSaving(true); setMessage('');
    try { await AppService().SavePreferences(form); setMessage('設定已儲存，下一次錄音或 AI 處理時生效。'); }
    catch (error) { setMessage(`儲存失敗：${String(error)}`); }
    finally { setSaving(false); }
  };

  return (
    <div style={{ flex: 1, overflow: 'auto', padding: 28 }}>
      <div style={{ maxWidth: 760, margin: '0 auto' }}>
        <h1 style={{ margin: 0, fontSize: 24, color: '#1e293b' }}>設定</h1>
        <p style={{ color: '#64748b' }}>個人設定會儲存在 Windows 使用者資料夾，更新程式時不會被覆蓋。</p>
        <div style={{ background: '#fff', border: '1px solid #e2e8f0', borderRadius: 12, padding: 22 }}>
          <div style={{ marginBottom: 18 }}><label style={labelStyle}>轉錄語言</label><select style={fieldStyle} value={form.language} onChange={e => update('language', e.target.value)}><option value="zh">中文</option><option value="en">英文</option><option value="ja">日文</option><option value="ko">韓文</option><option value="auto">自動偵測</option></select></div>
          <div style={{ marginBottom: 18 }}><label style={labelStyle}>Whisper 模型</label><select style={fieldStyle} value={form.whisperModel} onChange={e => update('whisperModel', e.target.value)}>{whisperModels.length === 0 && <option value={form.whisperModel}>{form.whisperModel}</option>}{whisperModels.map(model => <option key={model.path} value={model.path}>{model.name}</option>)}</select></div>
          <div style={{ marginBottom: 18 }}><label style={labelStyle}>Ollama 摘要模型</label><select style={fieldStyle} value={form.ollamaModel} onChange={e => update('ollamaModel', e.target.value)}>{ollamaModels.length === 0 && <option value={form.ollamaModel}>{form.ollamaModel}</option>}{ollamaModels.map(model => <option key={model.path} value={model.path}>{model.name}</option>)}</select></div>
          <div style={{ marginBottom: 18 }}><label style={labelStyle}>輸出資料夾</label><input style={fieldStyle} value={form.outputDir} onChange={e => update('outputDir', e.target.value)} /></div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 18 }}>
            <div><label style={labelStyle}>摘要創意程度：{form.temperature.toFixed(1)}</label><input style={{ width: '100%' }} type="range" min="0" max="1.5" step="0.1" value={form.temperature} onChange={e => update('temperature', Number(e.target.value))} /></div>
            <div><label style={labelStyle}>最大輸出 Token</label><input style={fieldStyle} type="number" min="256" max="32768" value={form.maxTokens} onChange={e => update('maxTokens', Number(e.target.value))} /></div>
          </div>
          {message && <div style={{ marginBottom: 14, color: message.startsWith('儲存失敗') ? '#b91c1c' : '#15803d', fontSize: 13 }}>{message}</div>}
          <button disabled={saving} onClick={() => void save()} style={{ padding: '10px 18px', border: 0, borderRadius: 8, background: '#2563eb', color: '#fff', fontWeight: 700, cursor: saving ? 'wait' : 'pointer' }}>{saving ? '儲存中…' : '儲存設定'}</button>
        </div>
      </div>
    </div>
  );
};
