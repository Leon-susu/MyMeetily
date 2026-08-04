import React from 'react';
import { useAppState } from '../hooks/useAppState';

interface Props { onStartMeeting: () => void; onOpenModels: () => void; }

export const HomePage: React.FC<Props> = ({ onStartMeeting, onOpenModels }) => {
  const { deps, appInfo } = useAppState();
  const cards = [
    { label: 'Whisper 引擎', ok: deps?.whisperBinary.ok, value: appInfo?.whisperModel || '檢查中' },
    { label: 'AI 摘要', ok: deps?.ollama.ok, value: appInfo?.llmModel || '檢查中' },
    { label: '本機處理', ok: true, value: '錄音與資料保留在電腦' },
  ];
  return (
    <div style={{ flex: 1, overflow: 'auto', padding: 28 }}>
      <div style={{ maxWidth: 980, margin: '0 auto' }}>
        <div style={{ padding: '30px 32px', borderRadius: 16, background: 'linear-gradient(135deg,#2563eb,#4f46e5)', color: '#fff' }}>
          <div style={{ fontSize: 13, opacity: .8, marginBottom: 8 }}>歡迎使用 MyMeetily</div>
          <h1 style={{ margin: 0, fontSize: 28 }}>開始一場可搜尋、可整理的會議</h1>
          <p style={{ margin: '12px 0 22px', opacity: .88 }}>本機錄音、語音轉文字，再由 Ollama 產生重點與待辦事項。</p>
          <button onClick={onStartMeeting} style={{ border: 0, borderRadius: 8, padding: '11px 18px', color: '#1d4ed8', background: '#fff', fontWeight: 700, cursor: 'pointer' }}>
            開始新會議
          </button>
        </div>

        <h2 style={{ fontSize: 17, color: '#334155', margin: '26px 0 12px' }}>系統概況</h2>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3,minmax(0,1fr))', gap: 14 }}>
          {cards.map(card => (
            <div key={card.label} style={{ padding: 18, border: '1px solid #e2e8f0', borderRadius: 12, background: '#fff' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: '#64748b', fontSize: 12 }}>
                <span style={{ width: 8, height: 8, borderRadius: '50%', background: card.ok ? '#22c55e' : '#f59e0b' }} />
                {card.label}
              </div>
              <div style={{ marginTop: 10, color: '#1e293b', fontWeight: 700, overflowWrap: 'anywhere' }}>{card.value}</div>
            </div>
          ))}
        </div>

        <div style={{ marginTop: 22, padding: 20, border: '1px solid #e2e8f0', borderRadius: 12, background: '#fff', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <div style={{ fontWeight: 700, color: '#334155' }}>需要調整速度或摘要品質？</div>
            <div style={{ marginTop: 5, color: '#64748b', fontSize: 13 }}>可依電腦效能切換 Whisper 與 Ollama 模型。</div>
          </div>
          <button onClick={onOpenModels} style={{ padding: '9px 14px', borderRadius: 8, border: '1px solid #bfdbfe', background: '#eff6ff', color: '#2563eb', cursor: 'pointer' }}>
            管理模型
          </button>
        </div>
      </div>
    </div>
  );
};
