import React, { createContext, useContext, useState, ReactNode } from 'react';
import { Poll } from '../lib/mockData';

interface PollsContextType {
  userPolls: Poll[];
  createPoll: (title: string, options: string[]) => void;
  closePoll: (pollId: string) => void;
}

const PollsContext = createContext<PollsContextType | undefined>(undefined);

export function PollsProvider({ children }: { children: ReactNode }) {
  const [userPolls, setUserPolls] = useState<Poll[]>([]);

  const createPoll = (title: string, options: string[]) => {
    const newPoll: Poll = {
      id: `user-poll-${Date.now()}`,
      title,
      options: options.map((text, index) => ({
        id: `option-${index}`,
        text,
        votes: 0,
      })),
      createdAt: new Date(),
      isLive: true,
      history: [],
    };

    setUserPolls(prev => [newPoll, ...prev]);
  };

  const closePoll = (pollId: string) => {
    setUserPolls(prev =>
      prev.map(poll =>
        poll.id === pollId ? { ...poll, isLive: false } : poll
      )
    );
  };

  return (
    <PollsContext.Provider
      value={{
        userPolls,
        createPoll,
        closePoll,
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
