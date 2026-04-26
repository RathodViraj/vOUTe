import React, { useEffect, useState } from 'react';
import { PollCard } from '../components/PollCard';
import { useSettings } from '../contexts/SettingsContext';
import { Card } from '../components/ui/card';
import { Bookmark, Trash2 } from 'lucide-react';
import { clearBookmarks, getBookmarks, getPollById, toggleBookmark } from '../lib/api';
import type { Poll } from '../lib/types';
import { toast } from 'sonner';
import { Button } from '../components/ui/button';

export function BookmarksPage() {
  const [bookmarkedPolls, setBookmarkedPolls] = useState<string[]>([]);
  const [bookmarkedPollsData, setBookmarkedPollsData] = useState<Poll[]>([]);
  const { showHistoricalData } = useSettings();

  useEffect(() => {
    const loadBookmarks = async () => {
      try {
        const ids = await getBookmarks();
        setBookmarkedPolls(ids);
        const polls = await Promise.all(ids.map((id) => getPollById(id)));
        setBookmarkedPollsData(
          polls.sort((a, b) => b.createdAt.getTime() - a.createdAt.getTime()),
        );
      } catch (error) {
        toast.error(error instanceof Error ? error.message : 'Failed to load bookmarks');
      }
    };

    loadBookmarks();
  }, []);

  const handleBookmarkToggle = async (pollId: string) => {
    try {
      await toggleBookmark(pollId, false);
      setBookmarkedPolls((prev) => prev.filter((id) => id !== pollId));
      setBookmarkedPollsData((prev) => prev.filter((poll) => poll.id !== pollId));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to update bookmark');
    }
  };

  const handleClearAllBookmarks = async () => {
    try {
      await clearBookmarks();
      setBookmarkedPolls([]);
      setBookmarkedPollsData([]);
      toast.success('All bookmarks cleared');
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to clear bookmarks');
    }
  };

  return (
    <div className="max-w-4xl mx-auto">
      <div className="mb-8">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-4xl font-bold mb-2">Bookmarks</h1>
            <p className="text-muted-foreground">
              Polls you've saved for later
            </p>
          </div>
          {bookmarkedPollsData.length > 0 && (
            <Button
              variant="outline"
              onClick={handleClearAllBookmarks}
              className="shrink-0"
            >
              <Trash2 className="w-4 h-4 mr-2" />
              Clear All
            </Button>
          )}
        </div>
      </div>

      {bookmarkedPollsData.length === 0 ? (
        <Card className="p-12 text-center">
          <div className="flex flex-col items-center gap-4">
            <div className="w-16 h-16 rounded-full bg-muted flex items-center justify-center">
              <Bookmark className="w-8 h-8 text-muted-foreground" />
            </div>
            <div>
              <h3 className="text-xl font-semibold mb-2">No bookmarks yet</h3>
              <p className="text-muted-foreground">
                Save polls to access them quickly later
              </p>
            </div>
          </div>
        </Card>
      ) : (
        <div className="space-y-6">
          {bookmarkedPollsData.map(poll => (
            <PollCard
              key={poll.id}
              poll={poll}
              isBookmarked={true}
              onBookmarkToggle={handleBookmarkToggle}
              showHistoricalDataByDefault={showHistoricalData}
            />
          ))}
        </div>
      )}
    </div>
  );
}
