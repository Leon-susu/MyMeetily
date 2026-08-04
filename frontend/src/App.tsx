import React, { useEffect, useReducer, useState } from 'react';
import { appReducer, initialState, AppStateContext, AppDispatchContext, useAppState } from './hooks/useAppState';
import { useWailsEvents } from './hooks/useWailsEvents';
import { TitleBar } from './components/TitleBar';
import { AppNavigation } from './components/AppNavigation';
import { HomePage } from './components/HomePage';
import { ModelsPage } from './components/ModelsPage';
import { SettingsPage } from './components/SettingsPage';
import { HistoryPage } from './components/HistoryPage';
import { DevicePanel } from './components/DevicePanel';
import { ControlPanel } from './components/ControlPanel';
import { StatusPanel } from './components/StatusPanel';
import { LiveView } from './components/LiveView';
import { ProgressView } from './components/ProgressView';
import { ResultView } from './components/ResultView';
import type { AppPage } from './types';

const STYLE: Record<string, React.CSSProperties> = {
  app: { display: 'flex', flexDirection: 'column', height: '100vh', background: '#f8fafc', color: '#334155', fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif" },
  body: { flex: 1, display: 'flex', minHeight: 0, overflow: 'hidden' },
  page: { flex: 1, display: 'flex', minWidth: 0, overflow: 'hidden' },
  meetingSidebar: { width: 300, padding: 16, flexShrink: 0, overflow: 'auto', borderRight: '1px solid #e2e8f0', background: '#f8fafc' },
  meetingContent: { flex: 1, padding: 16, display: 'flex', overflow: 'hidden' },
  centered: { flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', background: '#fff', borderRadius: 10, border: '1px solid #e2e8f0' },
};

function App() {
  const [state, dispatch] = useReducer(appReducer, initialState);
  return <AppStateContext.Provider value={state}><AppDispatchContext.Provider value={dispatch}><AppInner /></AppDispatchContext.Provider></AppStateContext.Provider>;
}

function MeetingPage() {
  const { phase } = useAppState();
  return (
    <div style={STYLE.page}>
      <aside style={STYLE.meetingSidebar}><DevicePanel /><ControlPanel /><StatusPanel /></aside>
      <main style={STYLE.meetingContent}>
        {phase === 'checking' && <div style={STYLE.centered}><div style={{ color: '#64748b' }}>正在初始化系統…</div></div>}
        {phase === 'ready' && <div style={STYLE.centered}><div style={{ fontSize: 34, marginBottom: 12 }}>🎙</div><div style={{ fontWeight: 700, color: '#475569' }}>準備就緒</div><div style={{ color: '#94a3b8', marginTop: 6, fontSize: 13 }}>選擇麥克風後即可開始錄音</div></div>}
        {phase === 'recording' && <LiveView />}
        {phase === 'processing' && <ProgressView />}
        {phase === 'result' && <ResultView />}
      </main>
    </div>
  );
}

function AppInner() {
  useWailsEvents();
  const { phase } = useAppState();
  const [activePage, setActivePage] = useState<AppPage>('home');
  const busy = phase === 'recording' || phase === 'processing';

  useEffect(() => {
    if (phase === 'recording' || phase === 'processing' || phase === 'result') setActivePage('meeting');
  }, [phase]);

  const navigate = (page: AppPage) => {
    if (busy && (page === 'history' || page === 'models' || page === 'settings')) return;
    setActivePage(page);
  };

  return (
    <div style={STYLE.app}>
      <TitleBar />
      <div style={STYLE.body}>
        <AppNavigation activePage={activePage} busy={busy} onNavigate={navigate} />
        {activePage === 'home' && <HomePage onStartMeeting={() => navigate('meeting')} onOpenModels={() => navigate('models')} />}
        {activePage === 'meeting' && <MeetingPage />}
        {activePage === 'history' && <HistoryPage />}
        {activePage === 'models' && <ModelsPage />}
        {activePage === 'settings' && <SettingsPage />}
      </div>
    </div>
  );
}

export default App;
