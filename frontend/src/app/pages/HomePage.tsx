import React, { useState } from 'react';
import { useOutletContext } from 'react-router';
import { PollCard } from '../components/PollCard';
import { mockPolls, mockBookmarkedPolls } from '../lib/mockData';
import { useSettings } from '../contexts/SettingsContext';
import { Tabs, TabsList, TabsTrigger } from '../components/ui/tabs';

interface OutletContext {
  availableVotes: number;
  setAvailableVotes: React.Dispatch<React.SetStateAction<number>>;
}

type FilterType = 'all' | 'live' | 'closed';

export function HomePage() {
  const [bookmarkedPolls, setBookmarkedPolls] = useState<string[]>(mockBookmarkedPolls);
  const [filter, setFilter] = useState<FilterType>('all');
  const { availableVotes, setAvailableVotes } = useOutletContext<OutletContext>();
  const { showHistoricalData } = useSettings();

  const handleBookmarkToggle = (pollId: string) => {
    setBookmarkedPolls(prev =>
      prev.includes(pollId)
        ? prev.filter(id => id !== pollId)
        : [...prev, pollId]
    );
  };

  const handleVoteSubmit = (voteCount: number) => {
    setAvailableVotes(prev => prev - voteCount);
  };

  const filteredPolls = mockPolls.filter(poll => {
    if (filter === 'all') return true;
    if (filter === 'live') return poll.isLive;
    if (filter === 'closed') return !poll.isLive;
    return true;
  });

  return (
    <div className="max-w-4xl mx-auto">
      <div className="mb-8">
        <h1 className="text-4xl font-bold mb-2">Active Polls</h1>
        <p className="text-muted-foreground">
          Vote on the topics that matter to you
        </p>
      </div>

      <div className="mb-6">
        <Tabs value={filter} onValueChange={(value) => setFilter(value as FilterType)}>
          <TabsList>
            <TabsTrigger value="all">All</TabsTrigger>
            <TabsTrigger value="live">Live</TabsTrigger>
            <TabsTrigger value="closed">Closed</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      <div className="space-y-6">
        {filteredPolls.map(poll => (
          <PollCard
            key={poll.id}
            poll={poll}
            isBookmarked={bookmarkedPolls.includes(poll.id)}
            onBookmarkToggle={handleBookmarkToggle}
            availableVotes={availableVotes}
            onVoteSubmit={handleVoteSubmit}
            showHistoricalDataByDefault={showHistoricalData}
          />
        ))}
      </div>
    </div>
  );
}