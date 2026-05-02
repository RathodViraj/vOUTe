import React, { useEffect, useState } from 'react';
import { Card } from '../components/ui/card';
import { CheckCircle2 } from 'lucide-react';
import { getPastVotes, getPollsWsUrl } from '../lib/api';
import { PollCard } from '../components/PollCard';
import type { Poll } from '../lib/types';
import { toast } from 'sonner';

export function PastVotesPage() {
  const [pastVotes, setPastVotes] = useState<Poll[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const loadPastVotes = async () => {
      try {
        setIsLoading(true);
        const pollsData = await getPastVotes();
        setPastVotes(pollsData);
      } catch (error) {
        toast.error(error instanceof Error ? error.message : 'Failed to load past votes');
      } finally {
        setIsLoading(false);
      }
    };

    loadPastVotes();
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

        setPastVotes((prev) =>
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
  }, []);

  return (
    <div className="max-w-4xl mx-auto">
      <div className="mb-8">
        <h1 className="text-4xl font-bold mb-2">Past Votes</h1>
        <p className="text-muted-foreground">
          Your voting history from the last 24 hours
        </p>
      </div>

      {isLoading ? (
        <Card className="p-12 text-center">
          <p className="text-muted-foreground">Loading your voting history...</p>
        </Card>
      ) : pastVotes.length === 0 ? (
        <Card className="p-12 text-center">
          <div className="flex flex-col items-center gap-4">
            <div className="w-16 h-16 rounded-full bg-muted flex items-center justify-center">
              <CheckCircle2 className="w-8 h-8 text-muted-foreground" />
            </div>
            <div>
              <h3 className="text-xl font-semibold mb-2">No votes yet</h3>
              <p className="text-muted-foreground">
                Your voting history will appear here
              </p>
            </div>
          </div>
        </Card>
      ) : (
        <div className="space-y-4">
          {pastVotes.map((poll) => (
            <PollCard
              key={poll.id}
              poll={poll}
              showChart
              showHistoricalDataByDefault
              readOnly
            />
          ))}
        </div>
      )}
    </div>
  );
}
