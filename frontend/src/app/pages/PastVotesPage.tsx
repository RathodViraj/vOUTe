import React from 'react';
import { Card } from '../components/ui/card';
import { Badge } from '../components/ui/badge';
import { mockUserVotes } from '../lib/mockData';
import { formatDistanceToNow } from 'date-fns';
import { CheckCircle2 } from 'lucide-react';

export function PastVotesPage() {
  return (
    <div className="max-w-4xl mx-auto">
      <div className="mb-8">
        <h1 className="text-4xl font-bold mb-2">Past Votes</h1>
        <p className="text-muted-foreground">
          Your voting history from the last 24 hours
        </p>
      </div>

      {mockUserVotes.length === 0 ? (
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
          {mockUserVotes.map((vote, index) => (
            <Card key={index} className="p-6 hover:shadow-lg transition-shadow">
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1">
                  <h3 className="text-lg font-semibold mb-2">{vote.pollTitle}</h3>
                  <div className="flex items-center gap-2 flex-wrap">
                    <Badge variant="secondary" className="text-sm">
                      Voted: {vote.selectedOption}
                    </Badge>
                    <Badge className="text-sm bg-indigo-600">
                      {vote.voteCount} vote{vote.voteCount > 1 ? 's' : ''}
                    </Badge>
                  </div>
                </div>
                <div className="text-sm text-muted-foreground text-right">
                  {formatDistanceToNow(vote.timestamp, { addSuffix: true })}
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
