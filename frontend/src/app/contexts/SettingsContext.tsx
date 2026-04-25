import React, { createContext, useContext, useState, ReactNode, useEffect } from 'react';

interface SettingsContextType {
  showHistoricalData: boolean;
  setShowHistoricalData: (value: boolean) => void;
  hasSeenHistoricalDataToast: boolean;
  setHasSeenHistoricalDataToast: (value: boolean) => void;
}

const SettingsContext = createContext<SettingsContextType | undefined>(undefined);

export function SettingsProvider({ children }: { children: ReactNode }) {
  const [showHistoricalData, setShowHistoricalData] = useState<boolean>(() => {
    const saved = localStorage.getItem('showHistoricalData');
    return saved ? JSON.parse(saved) : false;
  });

  const [hasSeenHistoricalDataToast, setHasSeenHistoricalDataToast] = useState<boolean>(() => {
    const saved = localStorage.getItem('hasSeenHistoricalDataToast');
    return saved ? JSON.parse(saved) : false;
  });

  useEffect(() => {
    localStorage.setItem('showHistoricalData', JSON.stringify(showHistoricalData));
  }, [showHistoricalData]);

  useEffect(() => {
    localStorage.setItem('hasSeenHistoricalDataToast', JSON.stringify(hasSeenHistoricalDataToast));
  }, [hasSeenHistoricalDataToast]);

  return (
    <SettingsContext.Provider
      value={{
        showHistoricalData,
        setShowHistoricalData,
        hasSeenHistoricalDataToast,
        setHasSeenHistoricalDataToast,
      }}
    >
      {children}
    </SettingsContext.Provider>
  );
}

export function useSettings() {
  const context = useContext(SettingsContext);
  if (context === undefined) {
    throw new Error('useSettings must be used within a SettingsProvider');
  }
  return context;
}
