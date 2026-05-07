import React, { createContext, useCallback, useContext, useState, ReactNode, useEffect } from 'react';
import { closePoll as closePollRequest, createPoll as createPollRequest, getMyPolls, getPollsWsUrl } from '../lib/api';
import { useAuth } from './AuthContext';
import type { Poll } from '../lib/types';

interface PollsContextType {
  userPolls: Poll[];
  createPoll: (title: string, options: string[]) => Promise<void>;
  closePoll: (pollId: string) => Promise<void>;
  refreshMyPolls: () => Promise<void>;
}

const PollsContext = createContext<PollsContextType | undefined>(undefined);

export function PollsProvider({ children }: { children: ReactNode }) {
  const [userPolls, setUserPolls] = useState<Poll[]>([]);
  const { isAuthenticated } = useAuth();

  const refreshMyPolls = useCallback(async () => {
    const polls = await getMyPolls();
    setUserPolls(polls);
  }, []);

  useEffect(() => {
    // Only establish WebSocket connection if user is authenticated
    if (!isAuthenticated) {
      return;
    }

    const ws = new WebSocket(getPollsWsUrl());

    ws.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data) as {
          changes?: Array<{
            poll_id: string;
            options: Array<{ option_id: string; vote_count: number }>;
          }>;
        };

        if (!payload.changes || payload.changes.length === 0) {
          return;
        }

        const now = new Date();

        setUserPolls((prev) =>
          prev.map((poll) => {
            const update = payload.changes?.find((c) => c.poll_id === poll.id);
            if (!update) return poll;

            const updatedOptions = poll.options.map((option) => {
              const optionUpdate = update.options.find((o) => o.option_id === option.id);
              if (!optionUpdate) return option;
              return { ...option, votes: optionUpdate.vote_count };
            });

            const historyPoint = update.options.map((optionUpdate) => ({
              timestamp: now,
              optionId: optionUpdate.option_id,
              votes: optionUpdate.vote_count,
            }));

            return {
              ...poll,
              options: updatedOptions,
              history: [...(poll.history || []), ...historyPoint].slice(-240),
            };
          }),
        );
      } catch {
        // Ignore malformed socket payloads.
      }
    };

    return () => {
      ws.close();
    };
  }, [isAuthenticated]);

  const createPoll = useCallback(async (title: string, options: string[]) => {
    await createPollRequest(title, options);
    await refreshMyPolls();
  }, [refreshMyPolls]);

  const closePoll = useCallback(async (pollId: string) => {
    await closePollRequest(pollId);
    await refreshMyPolls();
  }, [refreshMyPolls]);

  return (
    <PollsContext.Provider
      value={{
        userPolls,
        createPoll,
        closePoll,
        refreshMyPolls,
      }}
    >
      {children}
    </PollsContext.Provider>
  );
}

export function usePolls() {
  const context = useContext(PollsContext);
  if (context === undefined) {
    throw new Error('usePolls must be used within a PollsProvider');
  }
  return context;
}
