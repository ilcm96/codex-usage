export type UsageGlobalTotals = {
  totalTokens: number;
  inputTokens: number;
  outputTokens: number;
  costUsd: number;
  sessions: number;
  projects: number;
  devices: number;
  patchAddedLines: number;
};

export type Project = {
  id: string;
  displayName: string;
  cwd: string;
  relativePath: string;
  repository: string;
  repositoryUrl: string;
  sessions: number;
  totalTokens: number;
  costUsd: number;
};

export type ProjectChartItem = Project & {
  chartDetail: string;
  chartKey: string;
  chartLabel: string;
};

export type Session = {
  id: string;
  startedAt: string | null;
  updatedAt: string | null;
  cwd: string;
  branch: string;
  repository: string;
  repositoryUrl: string;
  project: string;
  device: string;
  title?: string;
  displayTitle?: string;
  displaySubtitle?: string;
  userIntent?: string;
  dominantLanguage?: string;
  firstUserMessage?: string;
  lastUserMessage?: string;
  shortSummary?: string;
  mainModel?: string;
  durationSeconds?: number;
  cacheHitRate?: number;
  conversationTurns?: number;
  searchableMessages?: number;
  searchableTools?: number;
  inputTokens: number;
  cachedInputTokens: number;
  cacheWriteInputTokens: number;
  outputTokens: number;
  reasoningOutputTokens: number;
  totalTokens: number;
  costUsd: number;
  models: string;
  messageCount?: number;
  userMessageCount?: number;
  assistantMessageCount?: number;
  toolCallCount?: number;
  patchAddedLines?: number;
};

export type UsageTotals = {
  inputTokens: number;
  cachedInputTokens: number;
  cacheWriteInputTokens: number;
  outputTokens: number;
  reasoningOutputTokens: number;
  totalTokens: number;
  costUsd: number;
  sessions: number;
  messages: number;
  toolCalls: number;
  patchAddedLines: number;
};

export type UsageSummary = {
  current: UsageTotals;
  activeDays: number;
  cacheHitRate: number;
  avgSessionCost: number;
};

export type UsageCalendarDay = {
  date: string;
  totalTokens: number;
  costUsd: number;
  projects: number;
};

export type SessionsListResponse = {
  items: Session[];
  limit: number;
  nextOffset: number;
  offset: number;
  total: number;
  totals: UsageTotals;
};

export type TimelineItem = {
  seq: number;
  occurredAt: string | null;
  kind: string;
  role: string;
  toolName: string;
  status: string;
  text: string;
};

export type SessionDetail = {
  session: Session & {
    commitHash: string;
    messageCount: number;
    toolCallCount: number;
    patchAddedLines: number;
  };
  models: ModelUsage[] | null;
  timeline: TimelineItem[] | null;
};

export type TimelineResponse = {
  items: TimelineItem[];
  limit: number;
  nextOffset: number;
  offset: number;
  total: number;
};

export type ReaderTurn = {
  turnIndex: number;
  userSeq: number | null;
  assistantSeq: number | null;
  startedAt: string | null;
  endedAt: string | null;
  userText: string;
  assistantText: string;
  toolCallCount: number;
  toolResultCount: number;
  toolNames: string[];
};

export type SessionReaderSummary = {
  sessionId: string;
  title: string;
  displayTitle: string;
  displaySubtitle: string;
  userIntent: string;
  dominantLanguage: string;
  firstUserMessage: string;
  lastUserMessage: string;
  shortSummary: string;
  messageCount: number;
  userMessageCount: number;
  assistantMessageCount: number;
  toolCallCount: number;
  mainModel: string;
  durationSeconds: number;
  cacheHitRate: number;
  startedAt: string | null;
  updatedAt: string | null;
  conversationTurnCount: number;
  searchableDocumentRows: number;
};

export type ReaderResponse = {
  summary: SessionReaderSummary;
  items: ReaderTurn[];
  limit: number;
  nextOffset: number;
  offset: number;
  total: number;
};

export type SearchResult = {
  kind: string;
  documentScope?: string;
  sessionId: string;
  seq: number;
  turnIndex?: number | null;
  occurredAt: string | null;
  role: string;
  toolName: string;
  title: string;
  text: string;
  snippet: string;
  rankWeight?: number;
  defaultSearchable?: boolean;
  matchStart: number;
  matchEnd: number;
  cwd: string;
  branch: string;
  repository: string;
  repositoryUrl: string;
  project: string;
  sessionTitle?: string;
  sessionSummary?: string;
};

export type SearchResponse = {
  items: SearchResult[];
  limit: number;
  nextOffset: number;
  offset: number;
  total: number;
  totalKnown?: boolean;
};

export type ModelUsage = {
  model: string;
  inputTokens: number;
  cachedInputTokens: number;
  cacheWriteInputTokens: number;
  outputTokens: number;
  reasoningOutputTokens: number;
  totalTokens: number;
  costUsd: number;
};

export type ArchiveStatus = {
  sessions: number;
  devices: number;
  rawBytes: number;
  sessionEvents: number;
  messages: number;
  toolEvents: number;
  usageEvents: number;
  missingRawFiles: number;
  missingRawSha: number;
  oldestIngestedAt: string | null;
  newestIngestedAt: string | null;
  oldestSessionTime: string | null;
  newestSessionTime: string | null;
};

export type ArchiveBreakdown = {
  id: string;
  name: string;
  hostname?: string;
  url?: string;
  sessions: number;
  rawBytes: number;
  lastIngestedAt: string | null;
};

export type ArchiveIntegrity = {
  checked: number;
  ok: number;
  missingPath: number;
  missingFile: number;
  sizeMismatch: number;
  missingSha: number;
  issues: Array<{ sessionId: string; path: string; problem: string }>;
};

export type ArchiveHealth = {
  status: string;
  sessions: number;
  archiveRows: number;
  sessionSummaries: number;
  conversationTurns: number;
  searchDocuments: number;
  defaultSearchDocs: number;
  messages: number;
  toolEvents: number;
  verifiedArchiveRows: number;
  missingRawFiles: number;
  missingArchiveRows: number;
  latestIngestedAt: string | null;
  oldestIngestedAt: string | null;
};

export type UsageWindow = {
  label: string;
  days: number;
  from: string | null;
  to: string | null;
  totals: UsageTotals;
  cacheHitRate: number;
};

export type FacetOption = {
  id: string;
  label: string;
  detail: string;
  count: number;
};

export type Facets = {
  dateRange: {
    oldest: string;
    newest: string;
  };
  devices: FacetOption[];
  repositories: FacetOption[];
  projects: FacetOption[];
  models: FacetOption[];
  branches: FacetOption[];
};

export type UsageSeriesPoint = {
  bucket: string;
  inputTokens: number;
  cachedInputTokens: number;
  cacheWriteInputTokens: number;
  outputTokens: number;
  reasoningOutputTokens: number;
  totalTokens: number;
  costUsd: number;
  patchAddedLines: number;
};

export type UsageBreakdown = {
  id: string;
  label: string;
  detail: string;
  sessions: number;
  inputTokens: number;
  cachedInputTokens: number;
  cacheWriteInputTokens: number;
  outputTokens: number;
  reasoningOutputTokens: number;
  totalTokens: number;
  costUsd: number;
  patchAddedLines: number;
};

export type DashboardFilters = {
  from: string;
  to: string;
  deviceId: string;
  repositoryId: string;
  projectId: string;
  model: string;
};
