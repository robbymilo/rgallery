import React from 'react';

const Error: React.FC<{ error: string }> = ({ error }) => {
  return (
    <div className="flex min-h-[50vh] flex-col items-center justify-center p-8 text-center">
      <p className="text-red-600 dark:text-red-400">{error}</p>
    </div>
  );
};

export default Error;
