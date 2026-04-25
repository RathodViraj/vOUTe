import React from 'react';
import { useParams, useNavigate } from 'react-router';
import { Button } from '../components/ui/button';
import { Card } from '../components/ui/card';
import { Separator } from '../components/ui/separator';
import { ArrowLeft } from 'lucide-react';
import { mockPolls, mockComments } from '../lib/mockData';
import { formatDistanceToNow } from 'date-fns';
import { Badge } from '../components/ui/badge';

export function CommentsPage() {
  const { pollId } = useParams();
  const navigate = useNavigate();

  const poll = mockPolls.find(p => p.id === pollId);
  const comments = mockComments.filter(c => c.pollId === pollId);

  if (!poll) {
    return (
      <div className="max-w-4xl mx-auto text-center">
        <p className="text-muted-foreground">Poll not found</p>
      </div>
    );
  }

  const totalVotes = poll.options.reduce((sum, option) => sum + option.votes, 0);

  return (
    <div className="max-w-7xl mx-auto">
      <Button
        variant="ghost"
        onClick={() => navigate(-1)}
        className="mb-6"
      >
        <ArrowLeft className="w-4 h-4 mr-2" />
        Back
      </Button>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left Side - Sticky Poll Card */}
        <div className="lg:sticky lg:top-24 lg:self-start">
          <Card className="p-6 space-y-4">
            <div>
              <h2 className="text-2xl font-semibold mb-3">{poll.title}</h2>
              <div className="flex items-center gap-2">
                <Badge variant="secondary" className="text-xs">
                  {totalVotes.toLocaleString()} votes
                </Badge>
                {poll.isLive ? (
                  <Badge className="text-xs bg-green-500 hover:bg-green-600">
                    <span className="inline-block w-2 h-2 bg-white rounded-full mr-1.5 animate-pulse"></span>
                    Live
                  </Badge>
                ) : (
                  <Badge variant="secondary" className="text-xs text-muted-foreground">
                    Closed
                  </Badge>
                )}
              </div>
            </div>

            <Separator />

            <div className="space-y-3">
              <h3 className="text-sm font-medium text-muted-foreground">Results</h3>
              {poll.options.map(option => {
                const percentage = totalVotes > 0 ? (option.votes / totalVotes) * 100 : 0;
                
                return (
                  <div key={option.id} className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="font-medium">{option.text}</span>
                      <span className="text-muted-foreground">
                        {option.votes.toLocaleString()} ({percentage.toFixed(1)}%)
                      </span>
                    </div>
                    <div className="w-full h-2 bg-muted rounded-full overflow-hidden">
                      <div
                        className="h-full bg-indigo-600 rounded-full transition-all duration-500"
                        style={{ width: `${percentage}%` }}
                      />
                    </div>
                  </div>
                );
              })}
            </div>
          </Card>
        </div>

        {/* Right Side - Scrollable Comments */}
        <div className="space-y-4">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-xl font-semibold">
              Comments ({comments.length})
            </h3>
          </div>

          {comments.length === 0 ? (
            <Card className="p-12 text-center">
              <p className="text-muted-foreground">No comments yet. Be the first to comment!</p>
            </Card>
          ) : (
            <div className="space-y-3">
              {comments.map(comment => (
                <Card key={comment.id} className="p-4 hover:shadow-md transition-shadow">
                  <div className="flex items-start justify-between mb-2">
                    <span className="font-semibold">{comment.username}</span>
                    <span className="text-sm text-muted-foreground">
                      {formatDistanceToNow(comment.timestamp, { addSuffix: true })}
                    </span>
                  </div>
                  <p className="text-sm leading-relaxed">{comment.text}</p>
                </Card>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
