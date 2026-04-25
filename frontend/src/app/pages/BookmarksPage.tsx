import React, { useState } from 'react';
import { PollCard } from '../components/PollCard';
import { mockPolls, mockBookmarkedPolls } from '../lib/mockData';
import { useSettings } from '../contexts/SettingsContext';
import { Card } from '../components/ui/card';
import { Bookmark } from 'lucide-react';

export function BookmarksPage() {
  const [bookmarkedPolls, setBookmarkedPolls] = useState<string[]>(mockBookmarkedPolls);
  const { showHistoricalData } = useSettings();

  const handleBookmarkToggle = (pollId: string) => {
    setBookmarkedPolls(prev =>
      prev.filter(id => id !== pollId)
    );
  };

  const bookmarkedPollsData = mockPolls.filter(poll => 
    bookmarkedPolls.includes(poll.id)
  );

  return (
    <div className="max-w-4xl mx-auto">
      <div className="mb-8">
        <h1 className="text-4xl font-bold mb-2">Bookmarks</h1>
        <p className="text-muted-foreground">
          Polls you've saved for later
        </p>
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
