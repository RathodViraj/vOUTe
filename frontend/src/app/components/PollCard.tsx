import React, { useState, useRef } from 'react';
import { useNavigate } from 'react-router';
import { Card } from './ui/card';
import { Button } from './ui/button';
import { RadioGroup, RadioGroupItem } from './ui/radio-group';
import { Label } from './ui/label';
import { Badge } from './ui/badge';
import { Bookmark, BookmarkCheck, Minus, Plus, MessageSquare } from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import type { Poll } from '../lib/types';
import { format } from 'date-fns';
import { toast } from 'sonner';

interface PollCardProps {
  poll: Poll;
  isBookmarked?: boolean;
  onBookmarkToggle?: (pollId: string) => Promise<void> | void;
  showChart?: boolean;
  availableVotes?: number;
  onVoteSubmit?: (pollId: string, optionId: string, voteCount: number) => Promise<void>;
  showHistoricalDataByDefault?: boolean;
}

export function PollCard({
  poll,
  isBookmarked = false,
  onBookmarkToggle,
  showChart = true,
  availableVotes = 24,
  onVoteSubmit,
  showHistoricalDataByDefault = false,
}: PollCardProps) {
  const navigate = useNavigate();
  const [selectedOption, setSelectedOption] = useState<string>('');
  const [voteCount, setVoteCount] = useState<number>(1);
  const [localVotes, setLocalVotes] = useState(poll.options);
  const [isLoading, setIsLoading] = useState(false);
  const [isDataRevealed, setIsDataRevealed] = useState(false);
  const longPressTimerRef = useRef<NodeJS.Timeout | null>(null);

  const totalVotes = localVotes.reduce((sum, option) => sum + option.votes, 0);
  const shouldShowData = showHistoricalDataByDefault || isDataRevealed;

  React.useEffect(() => {
    setLocalVotes(poll.options);
  }, [poll.options]);

  const handleVoteCountChange = (delta: number) => {
    // Strictly prevent going beyond available votes
    const maxAllowed = Math.min(100, availableVotes);
    const newCount = Math.max(1, Math.min(maxAllowed, voteCount + delta));
    setVoteCount(newCount);
  };

  const handleVote = async () => {
    if (!poll.isLive) {
      toast.error('This poll is closed');
      return;
    }

    if (!selectedOption) {
      toast.error('Please select an option to vote');
      return;
    }

    if (voteCount > availableVotes) {
      toast.error('Not enough votes available');
      return;
    }

    setIsLoading(true);

    try {
      if (onVoteSubmit) {
        await onVoteSubmit(poll.id, selectedOption, voteCount);
      }

      const updatedVotes = localVotes.map(option =>
        option.id === selectedOption
          ? { ...option, votes: option.votes + voteCount }
          : option
      );
      setLocalVotes(updatedVotes);

      toast.success(`Successfully voted with ${voteCount} vote${voteCount > 1 ? 's' : ''}!`);
      setSelectedOption('');
      setVoteCount(1);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to submit vote');
    } finally {
      setIsLoading(false);
    }
  };

  // Prepare chart data
  const chartData = poll.history
    .reduce((acc: any[], item) => {
      const existingEntry = acc.find(
        entry => entry.timestamp.getTime() === item.timestamp.getTime()
      );
      
      const option = poll.options.find(opt => String(opt.id) === String(item.optionId));
      const optionName = option?.text || item.optionId;
      
      if (existingEntry) {
        existingEntry[optionName] = item.votes;
      } else {
        acc.push({
          timestamp: item.timestamp,
          time: format(item.timestamp, 'HH:mm'),
          [optionName]: item.votes,
        });
      }
      
      return acc;
    }, [])
    .sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());

  const colors = ['#6366f1', '#8b5cf6', '#ec4899', '#f59e0b'];

  const handleDoubleClick = () => {
    if (!showHistoricalDataByDefault) {
      setIsDataRevealed(prev => !prev);
    }
  };

  const handleTouchStart = () => {
    if (!showHistoricalDataByDefault) {
      longPressTimerRef.current = setTimeout(() => {
        setIsDataRevealed(prev => !prev);
      }, 500);
    }
  };

  const handleTouchEnd = () => {
    if (longPressTimerRef.current) {
      clearTimeout(longPressTimerRef.current);
      longPressTimerRef.current = null;
    }
  };

  return (
    <Card
      className="p-6 space-y-4 hover:shadow-lg transition-shadow"
      onDoubleClick={handleDoubleClick}
      onTouchStart={handleTouchStart}
      onTouchEnd={handleTouchEnd}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1">
          <h3 className="text-xl font-semibold mb-2">{poll.title}</h3>
          <div className="flex items-center gap-2">
            {shouldShowData && (
              <Badge variant="secondary" className="text-xs">
                {totalVotes.toLocaleString()} votes
              </Badge>
            )}
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
        <div className="flex items-center gap-2 shrink-0">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate(`/comments/${poll.id}`)}
            className="shrink-0"
          >
            <MessageSquare className="h-5 w-5" />
          </Button>
          {onBookmarkToggle && (
            <Button
              variant="ghost"
              size="icon"
              onClick={() => onBookmarkToggle(poll.id)}
              className="shrink-0"
            >
              {isBookmarked ? (
                <BookmarkCheck className="h-5 w-5 text-indigo-600" />
              ) : (
                <Bookmark className="h-5 w-5" />
              )}
            </Button>
          )}
        </div>
      </div>

      <RadioGroup value={selectedOption} onValueChange={setSelectedOption}>
        <div className="space-y-2">
          {localVotes.map(option => {
            const percentage = totalVotes > 0 ? (option.votes / totalVotes) * 100 : 0;
            
            return (
              <div key={option.id} className="relative">
                <div className="flex items-center space-x-3 p-3 rounded-lg border hover:bg-accent/50 transition-colors cursor-pointer">
                  <RadioGroupItem value={option.id} id={option.id} />
                  <Label htmlFor={option.id} className="flex-1 cursor-pointer">
                    <div className="flex items-center justify-between">
                      <span>{option.text}</span>
                      {shouldShowData && (
                        <span className="text-sm text-muted-foreground ml-4">
                          {option.votes.toLocaleString()} ({percentage.toFixed(1)}%)
                        </span>
                      )}
                    </div>
                  </Label>
                </div>
                {shouldShowData && (
                  <div
                    className="absolute bottom-0 left-0 h-1 bg-indigo-500 rounded-full transition-all duration-500"
                    style={{ width: `${percentage}%` }}
                  />
                )}
              </div>
            );
          })}
        </div>
      </RadioGroup>

{voteCount > 0 && selectedOption && voteCount <= availableVotes && (
        <div className="text-sm text-muted-foreground px-1">
          <p>After voting, you will have <span className="font-semibold text-indigo-600">{availableVotes - voteCount}</span> vote{availableVotes - voteCount !== 1 ? 's' : ''} left</p>
        </div>
      )}

      <div className="flex items-center gap-3 pt-2">
        <div className="flex items-center border rounded-lg">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => handleVoteCountChange(-1)}
            disabled={voteCount <= 1 || isLoading || !poll.isLive}
            className="h-9 w-9"
          >
            <Minus className="h-4 w-4" />
          </Button>
          <div className="px-4 py-2 min-w-[60px] text-center font-medium">
            {voteCount}
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => handleVoteCountChange(1)}
            disabled={voteCount >= Math.min(100, availableVotes) || isLoading || !poll.isLive}
            className="h-9 w-9"
          >
            <Plus className="h-4 w-4" />
          </Button>
        </div>
        <Button
          onClick={handleVote}
          disabled={!selectedOption || isLoading || availableVotes === 0 || voteCount > availableVotes || !poll.isLive}
          className="flex-1 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isLoading ? 'Voting...' : !poll.isLive ? 'Poll Closed' : availableVotes === 0 ? 'No Votes Available' : 'Vote'}
        </Button>
      </div>

      {showChart && shouldShowData && poll.history.length > 0 && (
        <div className="pt-4">
          <h4 className="text-sm font-medium mb-3 text-muted-foreground">Voting Trends (Last 24h)</h4>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
              <XAxis
                dataKey="time"
                className="text-xs"
                tick={{ fontSize: 12 }}
              />
              <YAxis
                className="text-xs"
                tick={{ fontSize: 12 }}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'hsl(var(--background))',
                  border: '1px solid hsl(var(--border))',
                  borderRadius: '6px',
                }}
              />
              <Legend wrapperStyle={{ fontSize: '12px' }} />
              {poll.options.map((option, index) => (
                <Line
                  key={option.id}
                  type="monotone"
                  dataKey={option.text}
                  stroke={colors[index % colors.length]}
                  strokeWidth={2}
                  dot={false}
                />
              ))}
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </Card>
  );
}