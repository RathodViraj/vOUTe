import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Card } from '../components/ui/card';
import { FileText } from 'lucide-react';
import { toast } from 'sonner';
import { PollCard } from '../components/PollCard';
import { closePoll, deletePoll, getMyPollsPage } from '../lib/api';
import type { Poll } from '../lib/types';

export function MyPollsPage() {
  const [userPolls, setUserPolls] = useState<Poll[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const sentinelRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const load = async () => {
      try {
        const page = await getMyPollsPage({ take: 20 });
        setUserPolls(page.items);
        setNextCursor(page.nextCursor);
        setHasMore(Boolean(page.nextCursor));
      } catch (error) {
        toast.error(error instanceof Error ? error.message : 'Failed to load your polls');
      }
    };

    load();
  }, []);

  const loadMore = useCallback(async () => {
    if (isLoadingMore || !hasMore || !nextCursor) return;
    setIsLoadingMore(true);

    try {
      const page = await getMyPollsPage({ cursor: nextCursor, take: 20 });
      setUserPolls((prev) => [...prev, ...page.items]);
      setNextCursor(page.nextCursor);
      setHasMore(Boolean(page.nextCursor));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to load more polls');
    } finally {
      setIsLoadingMore(false);
    }
  }, [hasMore, isLoadingMore, nextCursor]);

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

  const handleClosePoll = async (pollId: string) => {
    try {
      await closePoll(pollId);
      setUserPolls((prev) => prev.map((poll) => (poll.id === pollId ? { ...poll, isLive: false } : poll)));
      toast.success('Poll closed successfully');
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to close poll');
    }
  };

  const handleDeletePoll = async (pollId: string) => {
    try {
      await deletePoll(pollId);
      setUserPolls((prev) => prev.filter((poll) => poll.id !== pollId));
      toast.success('Poll deleted successfully');
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to delete poll');
    }
  };

  return (
    <div className="max-w-4xl mx-auto">
      <div className="mb-8">
        <h1 className="text-4xl font-bold mb-2">My Polls</h1>
        <p className="text-muted-foreground">
          Manage your created polls
        </p>
      </div>

      {userPolls.length === 0 ? (
        <Card className="p-12 text-center">
          <div className="flex flex-col items-center gap-4">
            <div className="w-16 h-16 rounded-full bg-muted flex items-center justify-center">
              <FileText className="w-8 h-8 text-muted-foreground" />
            </div>
            <div>
              <h3 className="text-xl font-semibold mb-2">No polls created yet</h3>
              <p className="text-muted-foreground">
                Create your first poll using the button in the bottom right corner
              </p>
            </div>
          </div>
        </Card>
      ) : (
        <div className="space-y-6">
          {userPolls.map(poll => (
            <PollCard
              key={poll.id}
              poll={poll}
              showChart
              showHistoricalDataByDefault
              readOnly
              onClosePoll={handleClosePoll}
              onDeletePoll={handleDeletePoll}
            />
          ))}
          <div ref={sentinelRef} className="h-10" />
          {isLoadingMore && <p className="text-center text-sm text-muted-foreground">Loading more polls...</p>}
          {!hasMore && userPolls.length > 0 && <p className="text-center text-sm text-muted-foreground">No more polls</p>}
        </div>
      )}
    </div>
  );
}
