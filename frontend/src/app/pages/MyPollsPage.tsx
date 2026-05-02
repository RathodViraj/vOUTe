import React, { useEffect } from 'react';
import { Card } from '../components/ui/card';
import { usePolls } from '../contexts/PollsContext';
import { FileText } from 'lucide-react';
import { toast } from 'sonner';
import { PollCard } from '../components/PollCard';

export function MyPollsPage() {
  const { userPolls, closePoll, refreshMyPolls } = usePolls();

  useEffect(() => {
    refreshMyPolls();
  }, [refreshMyPolls]);

  const handleClosePoll = async (pollId: string) => {
    try {
      await closePoll(pollId);
      toast.success('Poll closed successfully');
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to close poll');
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
            />
          ))}
        </div>
      )}
    </div>
  );
}
