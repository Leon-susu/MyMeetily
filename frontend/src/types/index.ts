// =============================================================================
// MyMeetily — TypeScript 类型定义
// =============================================================================

// ---- Application Phase ----
export type AppPhase = 'checking' | 'ready' | 'recording' | 'processing' | 'result';

// ---- Device ----
export interface DeviceInfo {
  id: string;
  name: string;
  type: 'mic' | 'speaker';
}

// ---- Dependency Status ----
export interface CheckResult {
  ok: boolean;
  message: string;
}

export interface DependencyStatus {
  whisperBinary: CheckResult;
  whisperModel: CheckResult;
  ollama: CheckResult;
  summaryEnabled: boolean;
}

// ---- App Info ----
export interface AppInfo {
  engine: string;
  whisperModel: string;
  llmModel: string;
}

export type AppPage = 'home' | 'meeting' | 'history' | 'models' | 'settings';

export interface Preferences {
  whisperModel: string;
  ollamaModel: string;
  language: string;
  outputDir: string;
  temperature: number;
  maxTokens: number;
}

export interface ModelOption {
  name: string;
  path: string;
  size: number;
  active: boolean;
}

export interface ModelCatalogItem {
  id: string;
  kind: 'whisper' | 'ollama';
  name: string;
  path: string;
  size: number;
  description: string;
  installed: boolean;
  active: boolean;
  recommended: boolean;
}

export interface HardwareProfile {
  cpuThreads: number;
  architecture: string;
  gpus: string[];
  hasNvidia: boolean;
  recommendedModel: string;
  recommendation: string;
}

export interface ModelOperationProgress {
  kind: 'whisper' | 'ollama';
  modelId: string;
  phase: 'starting' | 'downloading' | 'done' | 'cancelled' | 'error';
  message: string;
  downloaded: number;
  total: number;
  percent: number;
}

export interface MeetingHistoryItem {
  id: string;
  title: string;
  sourceAudio: string;
  generatedAt: string;
  summary: string;
  htmlPath: string;
  markdownPath: string;
  transcriptPath: string;
}

// ---- Transcript Segment ----
export interface Segment {
  start: number;
  end: number;
  text: string;
  confidence: number;
}

// ---- Meeting State (from Go backend) ----
export interface MeetingState {
  audioFilePath: string;
  language: string;
  rawTranscript: string;
  meetingNotes: string;
  summaryContent: string;
  summaryError: string;
  summaryEnabled: boolean;
  audioDuration: number;
  outputPath: string;
  htmlOutputPath: string;
  transcriptPath: string;
  markdownPath: string;
  segments: Segment[];
  error: string;
}

// ---- Pipeline Progress ----
export interface StageStatus {
  name: string;
  status: 'pending' | 'active' | 'done';
}

// ---- Recording ----
export interface TranscriptLine {
  text: string;
  index: number;
}

// ---- Wails Event Payloads ----
export interface RecordStartedPayload {
  outputPath: string;
}

export interface RecordStoppedPayload {
  outputPath: string;
  duration: number;
}

export interface PeakLevelPayload {
  level: number;
  elapsed: number;
}

export interface TranscriptPayload {
  text: string;
}

export interface PipelineProgressPayload {
  stage: string;
}

export interface AppStatusPayload {
  phase: string;
  message: string;
  summaryEnabled?: boolean;
}
