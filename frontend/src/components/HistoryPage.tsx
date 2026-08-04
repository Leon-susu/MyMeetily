import React, { useEffect, useMemo, useState } from 'react';
import { AppService } from '../hooks/useWailsEvents';
import type { MeetingHistoryItem } from '../types';

export const HistoryPage: React.FC = () => {
  const [items, setItems] = useState<MeetingHistoryItem[]>([]);
  const [query, setQuery] = useState('');
  const [message, setMessage] = useState('正在讀取會議歷史…');

  const load = async () => {
    setMessage('正在讀取會議歷史…');
    try {
      const result = await AppService().ListMeetingHistory();
      setItems(result || []);
      setMessage(result?.length ? `共找到 ${result.length} 筆會議紀錄` : '目前還沒有會議紀錄');
    } catch (error) {
      setMessage(`讀取失敗：${String(error)}`);
    }
  };

  useEffect(() => { void load(); }, []);

  const open = async (action: 'report' | 'folder', path: string) => {
    try {
      if (action === 'report') {
        await AppService().OpenHistoryReport(path);
      } else {
        await AppService().OpenOutputFolder(path);
      }
    } catch (error) {
      setMessage(`開啟失敗：${String(error)}`);
    }
  };

  const filtered = useMemo(() => {
    const keyword = query.trim().toLocaleLowerCase('zh-TW');
    if (!keyword) return items;
    return items.filter(item => `${item.title} ${item.sourceAudio} ${item.summary}`.toLocaleLowerCase('zh-TW').includes(keyword));
  }, [items, query]);

  return (
    <div style={{ flex: 1, overflow: 'auto', padding: 28 }}>
      <div style={{ maxWidth: 920, margin: '0 auto' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'end', gap: 16, marginBottom: 18 }}>
          <div><h1 style={{ margin: 0, fontSize: 24, color: '#1e293b' }}>會議歷史</h1><p style={{ color: '#64748b', marginBottom: 0 }}>搜尋並重新開啟電腦裡已產生的會議報告。</p></div>
          <button onClick={() => void load()} style={secondaryButton}>重新整理</button>
        </div>
        <input value={query} onChange={event => setQuery(event.target.value)} placeholder="搜尋檔名或摘要內容…" style={{ width: '100%', boxSizing: 'border-box', padding: '11px 13px', borderRadius: 9, border: '1px solid #cbd5e1', marginBottom: 12, fontSize: 14 }} />
        <div style={{ marginBottom: 14, color: '#64748b', fontSize: 12 }}>{query ? `搜尋結果：${filtered.length} 筆` : message}</div>

        {filtered.length === 0 ? <div style={{ padding: 36, textAlign: 'center', borderRadius: 12, background: '#fff', border: '1px solid #e2e8f0', color: '#94a3b8' }}>沒有符合條件的會議紀錄。</div> : filtered.map(item => (
          <article key={item.id} style={{ padding: 18, marginBottom: 12, borderRadius: 12, background: '#fff', border: '1px solid #e2e8f0' }}>
            <div style={{ display: 'flex', alignItems: 'start', justifyContent: 'space-between', gap: 20 }}>
              <div style={{ minWidth: 0 }}>
                <h2 style={{ margin: 0, color: '#1e293b', fontSize: 17, overflow: 'hidden', textOverflow: 'ellipsis' }}>{item.title || '未命名會議'}</h2>
                <div style={{ marginTop: 5, color: '#94a3b8', fontSize: 12 }}>{new Date(item.generatedAt).toLocaleString('zh-TW')}</div>
                <p style={{ margin: '10px 0 0', color: '#64748b', fontSize: 13, lineHeight: 1.6 }}>{item.summary || '這筆紀錄沒有可預覽的摘要。'}</p>
              </div>
              <div style={{ display: 'flex', gap: 8, flexShrink: 0 }}>
                <button onClick={() => void open('report', item.htmlPath)} style={primaryButton}>開啟報告</button>
                <button onClick={() => void open('folder', item.htmlPath)} style={secondaryButton}>開啟資料夾</button>
              </div>
            </div>
          </article>
        ))}
      </div>
    </div>
  );
};

const primaryButton: React.CSSProperties = { borderRadius: 7, padding: '8px 12px', border: '1px solid #bfdbfe', background: '#eff6ff', color: '#2563eb', cursor: 'pointer' };
const secondaryButton: React.CSSProperties = { borderRadius: 7, padding: '8px 12px', border: '1px solid #cbd5e1', background: '#fff', color: '#475569', cursor: 'pointer' };
