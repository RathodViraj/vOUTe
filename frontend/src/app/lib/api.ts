import type { Comment, PastVoteItem, Poll, User, VoteHistory } from './types';

type ApiResponse<T> = {
  type: string;
  status: number;
  message: string;
  data: T;
  created_at: number;
};

type BackendOption = {
  id: string;
  text: string;
  vote_count?: number;
};

type BackendVote = {
  id: string;
  title: string;
  status: string;
  options: BackendOption[];
  created_at: number;
};

type HistoricOptionsData = {
  Timestamp: string;
  OptionID: string | number;
  VoteCount: number;
};

type HistoricDataResponse = {
  VoteID: string;
  OptionsData: HistoricOptionsData[];
};

const API_BASE = ((import.meta as any).env?.VITE_API_BASE_URL as string | undefined) || 'http://localhost:8080';
const ACCESS_TOKEN_KEY = 'voute_access_token';

export type PollSocketUpdate = {
  changes: Array<{
    poll_id: string;
    options: Array<{
      option_id: string;
      vote_count: number;
    }>;
  }>;
};

export function getPollsWsUrl(): string {
  const base = new URL(API_BASE);
  const wsProtocol = base.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${wsProtocol}//${base.host}/ws/polls`;
}

function getAccessToken(): string | null {
  // Keep token isolated per tab so multiple accounts can be used in parallel.
  // Fall back to localStorage once for migration from older builds.
  const tokenInSession = sessionStorage.getItem(ACCESS_TOKEN_KEY);
  if (tokenInSession) return tokenInSession;

  const tokenInLocal = localStorage.getItem(ACCESS_TOKEN_KEY);
  if (tokenInLocal) {
    sessionStorage.setItem(ACCESS_TOKEN_KEY, tokenInLocal);
    localStorage.removeItem(ACCESS_TOKEN_KEY);
  }

  return tokenInLocal;
}

export function setAccessToken(token: string) {
  sessionStorage.setItem(ACCESS_TOKEN_KEY, token);
}

export function clearAccessToken() {
  sessionStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(ACCESS_TOKEN_KEY);
}

export function hasAccessToken() {
  return Boolean(getAccessToken());
}

export async function refreshAccessToken() {
  const data = await request<{ access_token: string }>('/auth/refresh', {
    method: 'POST',
  });

  setAccessToken(data.access_token);
}

async function request<T>(
  path: string,
  options: RequestInit & { auth?: boolean } = {},
): Promise<T> {
  const headers = new Headers(options.headers || {});
  const hasBody = options.body !== undefined;

  if (hasBody && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  if (options.auth) {
    const token = getAccessToken();
    if (token) {
      headers.set('Authorization', `Bearer ${token}`);
    }
  }

  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
    credentials: 'include',
  });

  const json = (await response.json()) as ApiResponse<T>;
  if (!response.ok || json.type === 'error') {
    throw new Error(json?.message || 'Request failed');
  }

  return json.data;
}

function mapVoteToPoll(vote: BackendVote, history: VoteHistory[] = []): Poll {
  return {
    id: String(vote.id),
    title: vote.title,
    options: (vote.options || []).map((option) => ({
      id: String(option.id),
      text: option.text,
      votes: Number(option.vote_count || 0),
    })),
    createdAt: new Date((vote.created_at || 0) * 1000),
    isLive: vote.status === 'live',
    history,
  };
}

function mapHistory(data: HistoricDataResponse[] | undefined): Record<string, VoteHistory[]> {
  const byVoteId: Record<string, VoteHistory[]> = {};
  for (const voteData of data || []) {
    const voteId = String(voteData.VoteID);
    byVoteId[voteId] = (voteData.OptionsData || []).map((item) => ({
      timestamp: new Date(item.Timestamp),
      optionId: String(item.OptionID),
      votes: Number(item.VoteCount || 0),
    }));
  }
  return byVoteId;
}

async function attachHistory(votes: BackendVote[]): Promise<Poll[]> {
  if ((votes || []).length === 0) return [];

  let historyByVoteId: Record<string, VoteHistory[]> = {};
  try {
    historyByVoteId = await getHistoricData(votes.map((vote) => String(vote.id)));
  } catch {
    // Keep page usable if history endpoint is temporarily unavailable.
    historyByVoteId = {};
  }

  return votes
    .map((vote) => mapVoteToPoll(vote, historyByVoteId[String(vote.id)] || []))
    .sort((a, b) => b.createdAt.getTime() - a.createdAt.getTime());
}

export async function login(identifier: string, password: string) {
  const isEmail = identifier.includes('@');
  const path = isEmail ? '/auth/login?type=email' : '/auth/login?type=username';
  const body = isEmail
    ? { email: identifier, password }
    : { username: identifier, password };

  const data = await request<{ access_token: string }>(path, {
    method: 'POST',
    body: JSON.stringify(body),
  });

  setAccessToken(data.access_token);
}

export async function logout() {
  try {
    await request<null>('/auth/logout', {
      method: 'POST',
      auth: true,
    });
  } finally {
    clearAccessToken();
  }
}

export async function getCurrentUser(): Promise<User> {
  const user = await request<{ id: string; username: string; email: string }>('/users/me', {
    method: 'GET',
    auth: true,
  });

  return {
    id: String(user.id),
    username: user.username,
    email: user.email,
  };
}

export async function signup(username: string, email: string, password: string) {
  await request('/users/create', {
    method: 'POST',
    body: JSON.stringify({ username, email, password, role: 'user' }),
  });
}

export async function checkUsernameAvailability(username: string): Promise<boolean> {
  const response = await fetch(`${API_BASE}/users/check?username=${encodeURIComponent(username)}`);
  const json = (await response.json()) as ApiResponse<null>;
  if (!response.ok || json.type === 'error') {
    throw new Error(json?.message || 'Failed to check username');
  }

  return json.message.toLowerCase().includes('does not exist');
}

export async function resetPassword(email: string, newPassword: string) {
  await request('/auth/reset-password', {
    method: 'POST',
    body: JSON.stringify({ email, new_password: newPassword }),
  });
}

export async function updatePassword(email: string, newPassword: string) {
  await request('/users/updatePassword', {
    method: 'PUT',
    body: JSON.stringify({ email, new_password: newPassword }),
  });
}

export async function deleteAccount() {
  await request('/users/delete', {
    method: 'DELETE',
    auth: true,
  });
}

export async function getPolls(status?: 'live' | 'closed'): Promise<Poll[]> {
  const query = status ? `?status=${status}` : '';
  const votes = await request<BackendVote[]>(`/polls${query}`, { method: 'GET' });

  return attachHistory(votes || []);
}

export async function getPollById(pollId: string): Promise<Poll> {
  const vote = await request<BackendVote>(`/polls/${pollId}`, { method: 'GET' });
  let historyByVoteId: Record<string, VoteHistory[]> = {};
  try {
    historyByVoteId = await getHistoricData([pollId]);
  } catch {
    historyByVoteId = {};
  }

  return mapVoteToPoll(vote, historyByVoteId[String(vote.id)] || []);
}

export async function getMyPolls(): Promise<Poll[]> {
  const votes = await request<BackendVote[]>('/polls/creator', {
    method: 'GET',
    auth: true,
  });

  return attachHistory(votes || []);
}

export async function createPoll(title: string, options: string[]) {
  await request('/polls/create', {
    method: 'POST',
    auth: true,
    body: JSON.stringify({
      vote: { title },
      options: options.map((text) => ({ text })),
    }),
  });
}

export async function closePoll(pollId: string) {
  await request(`/polls/${pollId}`, {
    method: 'PATCH',
    auth: true,
  });
}

export async function voteOnPoll(pollId: string, optionId: string, count: number) {
  await request('/polls/update', {
    method: 'PUT',
    auth: true,
    body: JSON.stringify({ id: pollId, option_id: optionId, count }),
  });
}

export async function getRemainingVotes(): Promise<number> {
  const data = await request<{ remaining_votes: number }>('/polls/remaining', {
    method: 'GET',
    auth: true,
  });

  return Number(data.remaining_votes || 0);
}

export async function getHistoricData(ids: string[]): Promise<Record<string, VoteHistory[]>> {
  if (ids.length === 0) return {};

  const response = await request<HistoricDataResponse[]>('/polls/getHistoricData', {
    method: 'POST',
    body: JSON.stringify({ ids: ids }),
  });

  return mapHistory(response);
}

export async function getBookmarks(): Promise<string[]> {
  const bookmarks = await request<Array<{ vote_id: string }>>('/bookmarks', {
    method: 'GET',
    auth: true,
  });

  return (bookmarks || []).map((bookmark) => String(bookmark.vote_id));
}

export async function toggleBookmark(voteId: string, flag: boolean) {
  await request('/bookmarks/change', {
    method: 'PUT',
    auth: true,
    body: JSON.stringify({ vote_id: voteId, flag }),
  });
}

export async function clearBookmarks() {
  await request('/bookmarks', {
    method: 'DELETE',
    auth: true,
  });
}

export async function getComments(voteId: string): Promise<Comment[]> {
  const comments = await request<Array<{ id: string; vote_id: string; username: string; content: string; created_at: number }>>(
    `/comments/vote/${voteId}`,
    { method: 'GET' },
  );

  return (comments || []).map((comment) => ({
    id: String(comment.id),
    pollId: String(comment.vote_id),
    username: comment.username,
    text: comment.content,
    timestamp: new Date((comment.created_at || 0) * 1000),
  }));
}

export async function createComment(voteId: string, username: string, content: string): Promise<Comment> {
  const comment = await request<{ id: string; vote_id: string; username: string; content: string; created_at: number }>(
    '/comments/',
    {
      method: 'POST',
      auth: true,
      body: JSON.stringify({ vote_id: voteId, username, content }),
    },
  );

  return {
    id: String(comment.id),
    pollId: String(comment.vote_id),
    username: comment.username,
    text: comment.content,
    timestamp: new Date((comment.created_at || 0) * 1000),
  };
}

export async function getPastVotes(): Promise<PastVoteItem[]> {
  const votes = await request<BackendVote[]>('/polls/voted', {
    method: 'GET',
    auth: true,
  });

  return (votes || [])
    .map((vote) => ({
      pollId: String(vote.id),
      pollTitle: vote.title,
      timestamp: new Date((vote.created_at || 0) * 1000),
      isLive: vote.status === 'live',
    }))
    .sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime());
}
