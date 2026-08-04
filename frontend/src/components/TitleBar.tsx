import React from 'react';
import { useAppState } from '../hooks/useAppState';

export const TitleBar: React.FC = () => {
  const state = useAppState();
  const status = {
    checking: ['#f59e0b', '檢查中'], ready: ['#94a3b8', '就緒'],
    recording: ['#ef4444', '錄音中'], processing: ['#3b82f6', 'AI 處理中'], result: ['#10b981', '處理完成'],
  }[state.phase];
  return (
    <header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 22px', background: '#fff', borderBottom: '1px solid #e2e8f0', flexShrink: 0 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <div style={{ width: 30, height: 30, borderRadius: 7, background: '#3b82f6', color: '#fff', display: 'grid', placeItems: 'center', fontWeight: 800 }}>M</div>
        <div><span style={{ fontSize: 19, fontWeight: 800, color: '#1e293b' }}>MyMeetily</span><span style={{ marginLeft: 8, color: '#64748b', fontSize: 12 }}>本機離線 AI 會議助手</span></div>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: '#475569', fontSize: 13 }}>
        <span style={{ width: 8, height: 8, borderRadius: '50%', background: status[0] }} />{status[1]}
        {state.deps && <span style={{ marginLeft: 6, padding: '3px 8px', borderRadius: 5, border: `1px solid ${state.deps.summaryEnabled ? '#bbf7d0' : '#e2e8f0'}`, color: state.deps.summaryEnabled ? '#15803d' : '#64748b', background: state.deps.summaryEnabled ? '#f0fdf4' : '#f8fafc', fontSize: 11 }}>{state.deps.summaryEnabled ? 'AI 摘要' : '僅轉錄'}</span>}
      </div>
    </header>
  );
};
