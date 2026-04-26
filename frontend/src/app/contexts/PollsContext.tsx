import React, { createContext, useCallback, useContext, useState, ReactNode } from 'react';
import { closePoll as closePollRequest, createPoll as createPollRequest, getMyPolls } from '../lib/api';
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

  const refreshMyPolls = useCallback(async () => {
    const polls = await getMyPolls();
    setUserPolls(polls);
  }, []);

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
