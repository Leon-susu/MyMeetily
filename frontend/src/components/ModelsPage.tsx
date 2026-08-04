import React, { useEffect, useState } from 'react';
import { AppService } from '../hooks/useWailsEvents';
import type { HardwareProfile, ModelCatalogItem, ModelOperationProgress, Preferences } from '../types';

const formatSize = (bytes: number) => {
  if (!bytes) return '大小未知';
  const gb = bytes / 1024 / 1024 / 1024;
  return gb >= 1 ? `${gb.toFixed(1)} GB` : `${(bytes / 1024 / 1024).toFixed(0)} MB`;
};

export const ModelsPage: React.FC = () => {
  const [preferences, setPreferences] = useState<Preferences | null>(null);
  const [catalog, setCatalog] = useState<ModelCatalogItem[]>([]);
  const [hardware, setHardware] = useState<HardwareProfile | null>(null);
  const [message, setMessage] = useState('正在讀取模型…');
  const [busy, setBusy] = useState<string | null>(null);
  const [progress, setProgress] = useState<ModelOperationProgress | null>(null);

  const load = async () => {
    setMessage('正在讀取模型…');
    try {
      const [prefs, items, profile] = await Promise.all([
        AppService().GetPreferences(),
        AppService().ListModelCatalog(),
        AppService().GetHardwareProfile(),
      ]);
      setPreferences(prefs);
      setCatalog(items || []);
      setHardware(profile);
      setMessage('模型清單已更新');
    } catch (error) {
      setMessage(`讀取模型失敗：${String(error)}`);
    }
  };

  useEffect(() => {
    void load();
    const onProgress = (value: ModelOperationProgress) => {
      setProgress(value);
      setMessage(value.message);
      if (value.phase === 'done' || value.phase === 'cancelled' || value.phase === 'error') {
        setBusy(null);
        void load();
      }
    };
    window.runtime?.EventsOn('model:progress', onProgress);
    return () => window.runtime?.EventsOff('model:progress');
  }, []);

  const selectModel = async (model: ModelCatalogItem) => {
    if (!preferences) return;
    const next = {
      ...preferences,
      [model.kind === 'whisper' ? 'whisperModel' : 'ollamaModel']: model.path,
    };
    setBusy(`${model.kind}:${model.id}`);
    try {
      await AppService().SavePreferences(next);
      setPreferences(next);
      setMessage('模型已切換，下一次錄音或處理時生效。');
      await load();
    } catch (error) {
      setMessage(`切換失敗：${String(error)}`);
    } finally {
      setBusy(null);
    }
  };

  const installModel = async (model: ModelCatalogItem) => {
    setBusy(`${model.kind}:${model.id}`);
    setProgress({ kind: model.kind, modelId: model.id, phase: 'starting', message: '準備安裝…', downloaded: 0, total: model.size, percent: 0 });
    try {
      if (model.kind === 'whisper') await AppService().InstallWhisperModel(model.id);
      else await AppService().InstallOllamaModel(model.id);
      await load();
    } catch (error) {
      setMessage(`安裝失敗：${String(error)}`);
    } finally {
      setBusy(null);
    }
  };

  const deleteModel = async (model: ModelCatalogItem) => {
    if (!window.confirm(`確定要刪除 ${model.name}？下載的模型檔案會從這台電腦移除。`)) return;
    setBusy(`${model.kind}:${model.id}`);
    try {
      await AppService().DeleteModel(model.kind, model.id);
      setMessage(`${model.name} 已刪除`);
      await load();
    } catch (error) {
      setMessage(`刪除失敗：${String(error)}`);
    } finally {
      setBusy(null);
    }
  };

  const renderSection = (kind: 'whisper' | 'ollama', title: string, description: string) => (
    <section style={{ background: '#fff', border: '1px solid #e2e8f0', borderRadius: 12, padding: 20, marginBottom: 18 }}>
      <h2 style={{ margin: 0, color: '#1e293b', fontSize: 17 }}>{title}</h2>
      <p style={{ color: '#64748b', fontSize: 13 }}>{description}</p>
      {catalog.filter(model => model.kind === kind).map(model => {
        const modelBusy = busy === `${model.kind}:${model.id}`;
        return (
          <div key={`${model.kind}:${model.id}`} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 16, padding: '14px 4px', borderTop: '1px solid #f1f5f9' }}>
            <div style={{ minWidth: 0 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                <span style={{ color: '#334155', fontWeight: 700 }}>{model.name}</span>
                {model.recommended && <span style={{ padding: '2px 7px', borderRadius: 999, background: '#eff6ff', color: '#2563eb', fontSize: 11 }}>本機建議</span>}
                {model.active && <span style={{ padding: '2px 7px', borderRadius: 999, background: model.installed ? '#f0fdf4' : '#fff7ed', color: model.installed ? '#15803d' : '#c2410c', fontSize: 11 }}>{model.installed ? '使用中' : '目前設定（檔案缺失）'}</span>}
              </div>
              <div style={{ color: '#64748b', fontSize: 12, marginTop: 4 }}>{model.description}</div>
              <div style={{ color: '#94a3b8', fontSize: 12, marginTop: 3 }}>{formatSize(model.size)} · {model.installed ? '已安裝' : '尚未安裝'}</div>
            </div>
            <div style={{ display: 'flex', gap: 8, flexShrink: 0 }}>
              {!model.installed && <button disabled={busy !== null} onClick={() => void installModel(model)} style={primaryButton}>{modelBusy ? '安裝中…' : '安裝'}</button>}
              {model.installed && !model.active && <button disabled={busy !== null} onClick={() => void selectModel(model)} style={primaryButton}>切換使用</button>}
              {model.installed && !model.active && <button disabled={busy !== null} onClick={() => void deleteModel(model)} style={dangerButton}>刪除</button>}
            </div>
          </div>
        );
      })}
    </section>
  );

  return (
    <div style={{ flex: 1, overflow: 'auto', padding: 28 }}>
      <div style={{ maxWidth: 920, margin: '0 auto' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 18 }}>
          <div><h1 style={{ margin: 0, fontSize: 24, color: '#1e293b' }}>模型管理</h1><p style={{ color: '#64748b', marginBottom: 0 }}>安裝、切換或移除本機語音與摘要模型。</p></div>
          <button disabled={busy !== null} onClick={() => void load()} style={secondaryButton}>重新整理</button>
        </div>

        {hardware && <div style={{ padding: 16, marginBottom: 16, borderRadius: 10, border: '1px solid #bfdbfe', background: '#eff6ff', color: '#1e3a8a' }}>
          <div style={{ fontWeight: 700 }}>硬體建議</div>
          <div style={{ fontSize: 13, marginTop: 5 }}>{hardware.recommendation}</div>
          <div style={{ fontSize: 11, marginTop: 7, color: '#64748b' }}>CPU 執行緒：{hardware.cpuThreads} · GPU：{hardware.gpus.length ? hardware.gpus.join('、') : '未取得資料'}</div>
        </div>}

        <div style={{ marginBottom: 14, padding: '10px 12px', borderRadius: 8, background: '#f8fafc', color: '#64748b', fontSize: 12 }}>
          {message}
          {busy && progress && progress.modelId === busy.split(':').slice(1).join(':') && <div style={{ marginTop: 9 }}>
            <div style={{ height: 8, borderRadius: 999, background: '#e2e8f0', overflow: 'hidden' }}><div style={{ width: `${Math.max(2, Math.min(100, progress.percent || 2))}%`, height: '100%', background: '#2563eb', transition: 'width .2s' }} /></div>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 5 }}><span>{progress.percent > 0 ? `${progress.percent.toFixed(1)}%` : '處理中'}</span><button onClick={() => void AppService().CancelModelOperation()} style={{ border: 0, background: 'transparent', color: '#dc2626', cursor: 'pointer' }}>取消</button></div>
          </div>}
        </div>

        {renderSection('whisper', '語音轉文字模型', '模型越大通常越準確，但 CPU 筆電的處理時間也會增加。')}
        {renderSection('ollama', 'AI 摘要模型', 'Ollama 模型負責整理摘要、決議與待辦事項。')}
      </div>
    </div>
  );
};

const primaryButton: React.CSSProperties = { borderRadius: 7, padding: '8px 12px', border: '1px solid #bfdbfe', background: '#eff6ff', color: '#2563eb', cursor: 'pointer' };
const secondaryButton: React.CSSProperties = { padding: '9px 13px', border: '1px solid #cbd5e1', borderRadius: 8, background: '#fff', cursor: 'pointer' };
const dangerButton: React.CSSProperties = { borderRadius: 7, padding: '8px 12px', border: '1px solid #fecaca', background: '#fff', color: '#dc2626', cursor: 'pointer' };
