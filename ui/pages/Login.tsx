import React, { useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import Lock from '../svg/lock.svg?react';
import User from '../svg/user.svg?react';
import ArrowRight from '../svg/arrow-right.svg?react';
import Logo from '../svg/logo.svg?react';
import Loading from '../svg/loading.svg?react';

const Login: React.FC = () => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const { checkAuth, isAuthenticated } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const redirectPath = searchParams.get('redirect') || '/';

  // Redirect if already logged in
  React.useEffect(() => {
    if (isAuthenticated) {
      navigate(redirectPath);
    }
  }, [isAuthenticated, navigate, redirectPath]);

  // Focus username input on mount
  React.useEffect(() => {
    const usernameInput = document.getElementById('username');
    if (usernameInput) {
      usernameInput.focus();
    }
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim() || !password.trim()) return;

    setIsLoading(true);
    setError(null);
    try {
      const formData = new URLSearchParams();
      formData.append('username', username);
      formData.append('password', password);

      const response = await fetch('/api/signin', {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: formData.toString(),
      });

      if (response.ok) {
        await checkAuth();
        navigate(redirectPath);
      } else {
        const errorText = await response.text();
        setError(errorText || 'Login failed. Please check your credentials.');
      }
    } catch (err: unknown) {
      console.error(err);
      if (
        err &&
        typeof err === 'object' &&
        'message' in err &&
        typeof (err as { message?: unknown }).message === 'string'
      ) {
        setError((err as { message: string }).message);
      } else {
        setError('Login failed.');
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="dark:bg-charcoal-900 flex min-h-screen bg-white transition-colors duration-300">
      <div className="relative flex w-full items-center justify-center overflow-hidden p-8 lg:w-1/2">
        <div className="relative z-10 w-full max-w-md space-y-4">
          <div className="text-center">
            <div className="group mb-1 inline-flex w-full items-center justify-center text-gray-900 dark:text-gray-100">
              <Logo />
            </div>
          </div>

          <form onSubmit={handleSubmit} className="h-75 space-y-6">
            <div className="space-y-2">
              <label htmlFor="username" className="dark:text-charcoal-300 text-sm font-medium text-gray-700">
                Username
              </label>
              <div className="relative">
                <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                  <User className="dark:text-charcoal-500 h-5 w-5 text-gray-400" />
                </div>
                <input
                  id="username"
                  type="text"
                  required
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className={`dark:bg-charcoal-800 dark:text-charcoal-200 dark:placeholder-charcoal-600 mt-1 block w-full rounded-lg border bg-gray-50 py-3 pr-3 pl-10 text-gray-900 placeholder-gray-400 transition-all focus:ring-2 focus:outline-none ${error ? 'border-red-500/50 ring-red-500/20' : 'dark:border-charcoal-800 focus:ring-primary-500 border-gray-300 focus:border-transparent'}`}
                  placeholder="username"
                />
              </div>
            </div>

            <div className="space-y-2">
              <label htmlFor="password" className="dark:text-charcoal-300 text-sm font-medium text-gray-700">
                Password
              </label>
              <div className="relative">
                <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                  <Lock className="dark:text-charcoal-500 h-5 w-5 text-gray-400" />
                </div>
                <input
                  id="password"
                  type="password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className={`dark:bg-charcoal-800 dark:text-charcoal-200 dark:placeholder-charcoal-600 mt-1 block w-full rounded-lg border bg-gray-50 py-3 pr-3 pl-10 text-gray-900 placeholder-gray-400 transition-all focus:ring-2 focus:outline-none ${error ? 'border-red-500/50 ring-red-500/20' : 'dark:border-charcoal-800 focus:ring-primary-500 border-gray-300 focus:border-transparent'}`}
                  placeholder="password"
                />
              </div>
            </div>

            <button
              type="submit"
              disabled={isLoading || !username.trim() || !password.trim()}
              className="group bg-primary-600 hover:bg-primary-700 hover:shadow-primary-500/30 flex w-full items-center justify-center rounded-lg px-4 py-3 font-semibold text-white shadow-lg transition-all duration-200 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {isLoading ? (
                <span className="flex items-center">
                  <Loading className="mr-3 -ml-1 h-5 w-5 animate-spin text-white" />
                  Signing in...
                </span>
              ) : (
                <span className="flex items-center">
                  Sign in <ArrowRight className="ml-2 h-6 w-6 transition-transform group-hover:translate-x-1" />
                </span>
              )}
            </button>

            {error && (
              <div className="flex animate-pulse items-center rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-600 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-400">
                {error}
              </div>
            )}
          </form>
        </div>
      </div>

      <div className="bg-charcoal-900 relative hidden lg:block lg:w-1/2">
        <img
          src="/dist/public/login.jpg"
          alt="rgallery login"
          className="absolute inset-0 h-full w-full object-cover"
        />
        <div className="absolute inset-0 flex flex-col justify-end p-12 text-white">
          <blockquote>
            <footer className="flex gap-3">
              <div className="bg-charcoal-800/80 border-charcoal-800 font-small ml-auto rounded-md border p-4 text-[12px] tracking-wider text-white/80">
                <div>Common kingfisher</div>
                <div className="font-bold">Slovenia</div>
              </div>
            </footer>
          </blockquote>
        </div>
      </div>
    </div>
  );
};

export default Login;
