import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useOutletContext } from 'react-router';
import { PollCard } from '../components/PollCard';
import { useSettings } from '../contexts/SettingsContext';
import { Tabs, TabsList, TabsTrigger } from '../components/ui/tabs';
import { getBookmarks, getPollsPage, getPollsWsUrl, toggleBookmark, voteOnPoll } from '../lib/api';
import type { Poll } from '../lib/types';
import { toast } from 'sonner';

interface OutletContext {
  availableVotes: number;
  setAvailableVotes: React.Dispatch<React.SetStateAction<number>>;
}

type FilterType = 'all' | 'live' | 'closed';

export function HomePage() {
  const [polls, setPolls] = useState<Poll[]>([]);
  const [bookmarkedPolls, setBookmarkedPolls] = useState<string[]>([]);
  const [filter, setFilter] = useState<FilterType>('all');
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const { availableVotes, setAvailableVotes } = useOutletContext<OutletContext>();
  const { showHistoricalData } = useSettings();
  const sentinelRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const load = async () => {
      try {
        setPolls([]);
        setNextCursor(null);
        setHasMore(true);
        const [page, loadedBookmarks] = await Promise.all([
          getPollsPage({
            status: filter === 'all' ? undefined : filter,
            take: 50,
          }),
          getBookmarks(),
        ]);
        setPolls(page.items);
        setNextCursor(page.nextCursor);
        setHasMore(Boolean(page.nextCursor));
        setBookmarkedPolls(loadedBookmarks);
      } catch (error) {
        toast.error(error instanceof Error ? error.message : 'Failed to load polls');
      }
    };

    load();
  }, [filter]);

  const loadMore = useCallback(async () => {
    if (isLoadingMore || !hasMore || !nextCursor) return;
    setIsLoadingMore(true);

    try {
      const page = await getPollsPage({
        status: filter === 'all' ? undefined : filter,
        cursor: nextCursor,
        take: 50,
      });

      setPolls((prev) => [...prev, ...page.items]);
      setNextCursor(page.nextCursor);
      setHasMore(Boolean(page.nextCursor));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to load more polls');
    } finally {
      setIsLoadingMore(false);
    }
  }, [filter, hasMore, isLoadingMore, nextCursor]);

  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) {
          loadMore();
        }
      },
      { threshold: 0.1 },
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [loadMore]);

  const bookmarkTimersRef = useRef<Record<string, ReturnType<typeof setTimeout> | null>>({});

  useEffect(() => {
    return () => {
      Object.values(bookmarkTimersRef.current).forEach((t) => {
        if (t) clearTimeout(t);
      });
    };
  }, []);

  useEffect(() => {
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

        setPolls((prev) =>
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
              history: [...poll.history, ...historyPoint].slice(-240),
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
  }, []);

  const handleBookmarkToggle = (pollId: string) => {
    const isBookmarked = bookmarkedPolls.includes(pollId);
    // Optimistic UI update
    setBookmarkedPolls((prev) => (isBookmarked ? prev.filter((id) => id !== pollId) : [...prev, pollId]));

    // Debounce backend call
    if (bookmarkTimersRef.current[pollId]) {
      clearTimeout(bookmarkTimersRef.current[pollId] as ReturnType<typeof setTimeout>);
    }

    const newFlag = !isBookmarked;
    bookmarkTimersRef.current[pollId] = setTimeout(async () => {
      try {
        await toggleBookmark(pollId, newFlag);
      } catch (err) {
        // Revert optimistic update on failure
        setBookmarkedPolls((prev) => (prev.includes(pollId) ? prev.filter((id) => id !== pollId) : [...prev, pollId]));
        toast.error(err instanceof Error ? err.message : 'Failed to update bookmark');
      } finally {
        bookmarkTimersRef.current[pollId] = null;
      }
    }, 1000);
  };

  const handleVoteSubmit = async (pollId: string, optionId: string, voteCount: number) => {
    await voteOnPoll(pollId, optionId, voteCount);
    setAvailableVotes(prev => prev - voteCount);
  };

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
        {polls.map(poll => (
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
        <div ref={sentinelRef} className="h-10" />
        {isLoadingMore && <p className="text-center text-sm text-muted-foreground">Loading more polls...</p>}
        {!hasMore && polls.length > 0 && <p className="text-center text-sm text-muted-foreground">No more polls</p>}
      </div>
    </div>
  );
}