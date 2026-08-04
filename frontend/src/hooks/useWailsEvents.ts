// =============================================================================
// MyMeetily — Wails 事件订阅 Hook
// =============================================================================

import { useEffect } from 'react';
import { useAppDispatch } from './useAppState';
import type {
  RecordStartedPayload, RecordStoppedPayload, PeakLevelPayload,
  TranscriptPayload, PipelineProgressPayload, AppStatusPayload,
  DependencyStatus, MeetingState, AppInfo, DeviceInfo, Preferences, ModelOption,
  ModelCatalogItem, HardwareProfile, MeetingHistoryItem
} from '../types';

// Wails runtime — these are available globally when running in Wails
declare global {
  interface Window {
    go: {
      main: {
        App: {
          Greet: (name: string) => Promise<string>;
        };
        AppService: {
          GetAppInfo: () => Promise<AppInfo>;
          GetPreferences: () => Promise<Preferences>;
          SavePreferences: (preferences: Preferences) => Promise<void>;
          ListWhisperModels: () => Promise<ModelOption[]>;
          ListOllamaModels: () => Promise<ModelOption[]>;
          ListModelCatalog: () => Promise<ModelCatalogItem[]>;
          GetHardwareProfile: () => Promise<HardwareProfile>;
          InstallWhisperModel: (modelId: string) => Promise<void>;
          InstallOllamaModel: (modelId: string) => Promise<void>;
          CancelModelOperation: () => Promise<void>;
          DeleteModel: (kind: string, modelId: string) => Promise<void>;
          ListMeetingHistory: () => Promise<MeetingHistoryItem[]>;
          OpenHistoryReport: (path: string) => Promise<void>;
          OpenOutputFolder: (path: string) => Promise<void>;
          CopyToClipboard: (text: string) => Promise<void>;
          OpenFileDialog: () => Promise<string>;
        };
        DeviceService: {
          ListMicDevices: () => Promise<DeviceInfo[]>;
          ListSpeakerDevices: () => Promise<DeviceInfo[]>;
        };
        RecordService: {
          StartRecording: (mic: string, speaker: string) => Promise<void>;
          PauseRecording: () => Promise<void>;
          ResumeRecording: () => Promise<void>;
          StopRecording: () => Promise<string>;
          IsRecording: () => Promise<boolean>;
          GetElapsed: () => Promise<number>;
        };
        PipelineService: {
          RunPipeline: (audioPath: string) => Promise<void>;
          RetrySummary: () => Promise<void>;
          ImportAudioFile: (filePath: string) => Promise<void>;
          CancelPipeline: () => Promise<void>;
        };
        ResultService: {
          GetState: () => Promise<MeetingState | null>;
        };
      };
    };
    runtime: {
      EventsOn: (event: string, callback: (...args: any[]) => void) => void;
      EventsOff: (event: string) => void;
      EventsOnce: (event: string, callback: (...args: any[]) => void) => void;
      EventsEmit: (event: string, data?: any) => void;
      BrowserOpenURL: (url: string) => void;
    };
  }
}

// Wails service proxies — use these from components
export const AppService = () => window.go.main.AppService;
export const DeviceService = () => window.go.main.DeviceService;
export const RecordService = () => window.go.main.RecordService;
export const PipelineService = () => window.go.main.PipelineService;
export const ResultService = () => window.go.main.ResultService;

/**
 * Subscribe to all Wails backend events and dispatch them to the app reducer.
 */
export function useWailsEvents() {
  const dispatch = useAppDispatch();

  useEffect(() => {
    // Only subscribe in Wails environment
    if (!window.runtime) return;

    // ---- App Status ----
    window.runtime.EventsOn('app:status', (payload: AppStatusPayload) => {
      if (payload.phase === 'ready') {
        dispatch({ type: 'SET_PHASE', phase: 'ready' });
      }
    });

    // ---- Dependency Check ----
    window.runtime.EventsOn('app:dependency', (payload: any) => {
      const deps: DependencyStatus = {
        whisperBinary: payload.whisperBinary || { ok: false, message: '未知' },
        whisperModel:  payload.whisperModel  || { ok: false, message: '未知' },
        ollama:        payload.ollama        || { ok: false, message: '未知' },
        summaryEnabled: payload.summaryEnabled ?? false,
      };
      dispatch({ type: 'SET_DEPS', deps, appInfo: null });
    });

    window.runtime.EventsOn('app:info', (payload: AppInfo) => {
      dispatch({
        type: 'SET_DEPS',
        deps: { whisperBinary: { ok: true, message: '' }, whisperModel: { ok: true, message: '' }, ollama: { ok: true, message: '' }, summaryEnabled: true },
        appInfo: payload,
      });
    });

    window.runtime.EventsOn('settings:changed', (payload: AppInfo) => {
      dispatch({ type: 'SET_APP_INFO', appInfo: payload });
    });

    // Startup events can be emitted before React subscribes to them. Query the
    // already-bound backend once after subscriptions are installed so the UI
    // cannot remain in the "checking" phase forever after a fast startup.
    void AppService().GetAppInfo().then((payload: AppInfo) => {
      dispatch({
        type: 'SET_DEPS',
        deps: {
          whisperBinary: { ok: true, message: '' },
          whisperModel: { ok: true, message: '' },
          ollama: { ok: true, message: '' },
          summaryEnabled: true,
        },
        appInfo: payload,
      });
      dispatch({ type: 'SET_PHASE', phase: 'ready' });
    }).catch(() => undefined);

    // ---- Recording ----
    window.runtime.EventsOn('record:started', (payload: RecordStartedPayload) => {
      dispatch({ type: 'RECORDING_STARTED', outputPath: payload.outputPath });
    });

    window.runtime.EventsOn('record:stopped', () => {
      dispatch({ type: 'RECORDING_STOPPED' });
    });

    window.runtime.EventsOn('record:paused', () => {
      dispatch({ type: 'RECORDING_PAUSED' });
    });

    window.runtime.EventsOn('record:resumed', () => {
      dispatch({ type: 'RECORDING_RESUMED' });
    });

    window.runtime.EventsOn('record:peaklevel', (payload: PeakLevelPayload) => {
      dispatch({ type: 'SET_PEAK_LEVEL', level: payload.level, elapsed: payload.elapsed });
    });

    window.runtime.EventsOn('record:transcript', (payload: TranscriptPayload) => {
      dispatch({ type: 'ADD_TRANSCRIPT', text: payload.text });
    });

    // ---- Pipeline ----
    window.runtime.EventsOn('pipeline:progress', (payload: PipelineProgressPayload) => {
      dispatch({ type: 'SET_PHASE', phase: 'processing' });
      dispatch({ type: 'SET_CURRENT_STAGE', stage: payload.stage });
    });

    window.runtime.EventsOn('pipeline:done', (payload: MeetingState) => {
      dispatch({ type: 'PIPELINE_DONE', meetingState: payload });
    });

    window.runtime.EventsOn('pipeline:error', (payload: { error: string }) => {
      dispatch({ type: 'PIPELINE_ERROR', error: payload.error });
    });

    window.runtime.EventsOn('pipeline:cancelled', (payload: { message: string }) => {
      dispatch({ type: 'PIPELINE_CANCELLED', message: payload.message });
    });

    // Cleanup
    return () => {
      if (!window.runtime) return;
      window.runtime.EventsOff('app:status');
      window.runtime.EventsOff('app:dependency');
      window.runtime.EventsOff('app:info');
      window.runtime.EventsOff('settings:changed');
      window.runtime.EventsOff('record:started');
      window.runtime.EventsOff('record:stopped');
      window.runtime.EventsOff('record:paused');
      window.runtime.EventsOff('record:resumed');
      window.runtime.EventsOff('record:peaklevel');
      window.runtime.EventsOff('record:transcript');
      window.runtime.EventsOff('pipeline:progress');
      window.runtime.EventsOff('pipeline:done');
      window.runtime.EventsOff('pipeline:error');
      window.runtime.EventsOff('pipeline:cancelled');
    };
  }, [dispatch]);
}
