import React from 'react';
import Spinner from '../svg/loading.svg?react';

const Loading: React.FC = () => {
  return (
    <div className="flex min-h-[50vh] flex-col items-center justify-center space-y-4">
      <Spinner className="text-primary-500 h-10 w-10 animate-spin" />
      <p className="dark:text-charcoal-400 animate-pulse text-gray-500">Loading...</p>
    </div>
  );
};

export default Loading;
