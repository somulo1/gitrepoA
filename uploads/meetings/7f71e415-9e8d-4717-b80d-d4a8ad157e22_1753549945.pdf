import React, { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Eye, EyeOff, Wallet } from 'lucide-react';
import { formatCurrency } from '@/lib/utils';

interface ChamaWalletCardProps {
  chama: {
    id: number;
    name: string;
    balance: number;
  };
  wallet?: {
    id: number;
    balance: number;
    currency: string;
    lastUpdated: string;
  } | null;
}

const ChamaWalletCard: React.FC<ChamaWalletCardProps> = ({ chama, wallet }) => {
  const [showBalance, setShowBalance] = useState(true);
  
  const toggleBalanceVisibility = () => {
    setShowBalance(prev => !prev);
  };
  
  const formattedBalance = wallet 
    ? formatCurrency(wallet.balance, wallet.currency) 
    : formatCurrency(chama.balance, 'KES');
    
  const lastUpdated = wallet 
    ? new Date(wallet.lastUpdated).toLocaleString('en-US', { 
        hour: 'numeric',
        minute: 'numeric',
        hour12: true,
        month: 'short',
        day: 'numeric',
      })
    : 'Not available';
  
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-lg flex justify-between items-center">
          <span>Chama Wallet</span>
          <Button variant="ghost" size="sm" className="h-8 w-8 p-0" onClick={toggleBalanceVisibility}>
            {showBalance ? <Eye className="h-4 w-4" /> : <EyeOff className="h-4 w-4" />}
          </Button>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="bg-gradient-to-r from-primary to-primary-light rounded-xl p-4 text-white">
          <div className="flex items-center mb-4">
            <div className="w-10 h-10 rounded-full bg-white/10 flex items-center justify-center mr-3">
              <Wallet className="h-5 w-5 text-white" />
            </div>
            <div>
              <p className="text-sm opacity-90">Group Funds</p>
              <h2 className="text-2xl font-bold">
                {showBalance ? formattedBalance : '••••••'}
              </h2>
            </div>
          </div>
          
          <div className="text-xs opacity-90 mb-4">
            Last updated: {lastUpdated}
          </div>
          
          <div className="grid grid-cols-2 gap-2">
            <Button size="sm" className="bg-white/20 hover:bg-white/30 text-white">
              View Transactions
            </Button>
            <Button size="sm" className="bg-white/20 hover:bg-white/30 text-white">
              Export Statement
            </Button>
          </div>
        </div>
        
        <div className="mt-4 space-y-4">
          <div className="bg-neutral-50 dark:bg-neutral-900 rounded-lg p-3 flex justify-between items-center">
            <div>
              <p className="text-sm font-medium">Savings Goal</p>
              <p className="text-xs text-neutral-500 dark:text-neutral-400">
                December 2023
              </p>
            </div>
            <div className="text-right">
              <p className="font-medium">KES 500,000</p>
              <p className="text-xs text-neutral-500 dark:text-neutral-400">
                {Math.round((wallet?.balance || chama.balance) / 500000 * 100)}% Achieved
              </p>
            </div>
          </div>
          
          {/* Progress Bar */}
          <div className="h-2 w-full bg-neutral-200 dark:bg-neutral-700 rounded overflow-hidden">
            <div 
              className="h-full bg-primary" 
              style={{ width: `${Math.min(Math.round((wallet?.balance || chama.balance) / 500000 * 100), 100)}%` }}
            ></div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};

export default ChamaWalletCard;
