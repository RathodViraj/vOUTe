import React from 'react';
import { Card } from '../components/ui/card';
import { Button } from '../components/ui/button';
import { Badge } from '../components/ui/badge';
import { usePolls } from '../contexts/PollsContext';
import { FileText, Lock } from 'lucide-react';
import { toast } from 'sonner';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { format } from 'date-fns';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '../components/ui/alert-dialog';

export function MyPollsPage() {
  const { userPolls, closePoll } = usePolls();

  const handleClosePoll = (pollId: string) => {
    closePoll(pollId);
    toast.success('Poll closed successfully');
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
          {userPolls.map(poll => {
            const totalVotes = poll.options.reduce((sum, option) => sum + option.votes, 0);

            // Prepare chart data
            const colors = ['#6366f1', '#8b5cf6', '#ec4899', '#f59e0b'];
            const chartData = poll.history
              .reduce((acc: any[], item) => {
                const existingEntry = acc.find(
                  entry => entry.timestamp.getTime() === item.timestamp.getTime()
                );

                const option = poll.options.find(opt => opt.id === item.optionId);
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

            return (
              <Card key={poll.id} className="p-6 space-y-4">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1">
                    <h3 className="text-xl font-semibold mb-2">{poll.title}</h3>
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
                  {poll.isLive && (
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button variant="outline" size="sm">
                          <Lock className="w-4 h-4 mr-2" />
                          Close Poll
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>Close this poll?</AlertDialogTitle>
                          <AlertDialogDescription>
                            Once closed, users will no longer be able to vote on this poll. This action cannot be undone.
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>Cancel</AlertDialogCancel>
                          <AlertDialogAction
                            onClick={() => handleClosePoll(poll.id)}
                            className="bg-indigo-600 hover:bg-indigo-700"
                          >
                            Close Poll
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  )}
                </div>

                <div className="space-y-2">
                  {poll.options.map(option => {
                    const percentage = totalVotes > 0 ? (option.votes / totalVotes) * 100 : 0;

                    return (
                      <div key={option.id} className="relative">
                        <div className="flex items-center justify-between p-3 rounded-lg border">
                          <span>{option.text}</span>
                          <span className="text-sm text-muted-foreground ml-4">
                            {option.votes.toLocaleString()} ({percentage.toFixed(1)}%)
                          </span>
                        </div>
                        <div
                          className="absolute bottom-0 left-0 h-1 bg-indigo-500 rounded-full transition-all duration-500"
                          style={{ width: `${percentage}%` }}
                        />
                      </div>
                    );
                  })}
                </div>

                {poll.history.length > 0 && (
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
          })}
        </div>
      )}
    </div>
  );
}
