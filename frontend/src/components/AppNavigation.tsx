import React from 'react';
import type { AppPage } from '../types';

interface Props {
  activePage: AppPage;
  busy: boolean;
  onNavigate: (page: AppPage) => void;
}

const items: Array<{ page: AppPage; icon: string; label: string }> = [
  { page: 'home', icon: '⌂', label: '首頁' },
  { page: 'meeting', icon: '●', label: '會議錄音' },
  { page: 'history', icon: '▤', label: '會議歷史' },
  { page: 'models', icon: '◇', label: '模型管理' },
  { page: 'settings', icon: '⚙', label: '設定' },
];

export const AppNavigation: React.FC<Props> = ({ activePage, busy, onNavigate }) => (
  <nav style={{ width: 188, padding: '18px 12px', borderRight: '1px solid #e2e8f0', background: '#ffffff' }}>
    <div style={{ padding: '0 10px 12px', fontSize: 11, color: '#94a3b8', fontWeight: 700, letterSpacing: 1 }}>
      功能選單
    </div>
    {items.map(item => {
      const disabled = busy && (item.page === 'history' || item.page === 'models' || item.page === 'settings');
      const active = activePage === item.page;
      return (
        <button
          key={item.page}
          type="button"
          disabled={disabled}
          onClick={() => onNavigate(item.page)}
          style={{
            width: '100%', display: 'flex', alignItems: 'center', gap: 10,
            padding: '11px 12px', marginBottom: 6, borderRadius: 8,
            border: 'none', cursor: disabled ? 'not-allowed' : 'pointer',
            background: active ? '#eff6ff' : 'transparent',
            color: disabled ? '#cbd5e1' : active ? '#2563eb' : '#475569',
            fontSize: 14, fontWeight: active ? 700 : 500, textAlign: 'left',
          }}
        >
          <span style={{ width: 20, textAlign: 'center' }}>{item.icon}</span>
          {item.label}
        </button>
      );
    })}
    {busy && (
      <div style={{ margin: '18px 10px 0', padding: 10, borderRadius: 8, background: '#fff7ed', color: '#9a3412', fontSize: 11, lineHeight: 1.5 }}>
        錄音或 AI 處理期間暫停開啟歷史、模型與設定。
      </div>
    )}
  </nav>
);
