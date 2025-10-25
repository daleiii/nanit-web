'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import useSWR from 'swr'
import MainLayout from '@/components/layout/MainLayout'
import BabyCard from '@/components/baby/BabyCard'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import ErrorMessage from '@/components/ui/ErrorMessage'
import { api } from '@/lib/api'
import { useStatus } from '@/hooks/useStatus'
import type { AuthStatusResponse, WebAuthStatusResponse } from '@/types/api'

export default function Dashboard() {
  const router = useRouter()
  const [showPasswordLogin, setShowPasswordLogin] = useState(false)
  const [password, setPassword] = useState('')
  const [loginError, setLoginError] = useState('')
  const [isLoggingIn, setIsLoggingIn] = useState(false)
  
  // Check web authentication status first
  const { data: webAuthStatus, mutate: mutateWebAuth } = useSWR<WebAuthStatusResponse>(
    '/webauth/status',
    () => api.getWebAuthStatus(),
    {
      revalidateOnFocus: false,
      revalidateOnReconnect: false,
    }
  )

  // Check Nanit authentication status (only if web auth passes)
  const { data: authStatus } = useSWR<AuthStatusResponse>(
    webAuthStatus?.authenticated ? '/auth/status' : null,
    () => api.getAuthStatus(),
    {
      revalidateOnFocus: false,
      revalidateOnReconnect: false,
    }
  )

  // Only fetch baby data if both authentications pass
  const { babies, isLoading, isError } = useStatus(
    webAuthStatus?.authenticated && authStatus?.authenticated
  )

  useEffect(() => {
    if (webAuthStatus) {
      // If password protection is enabled but user not authenticated, show login
      if (webAuthStatus.password_protection_enabled && !webAuthStatus.authenticated) {
        setShowPasswordLogin(true)
        return
      }
      
      // If web auth passes but Nanit auth fails, redirect to setup
      if (webAuthStatus.authenticated && authStatus && !authStatus.authenticated) {
        router.push('/setup')
      }
    }
  }, [webAuthStatus, authStatus, router])

  const handlePasswordLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoggingIn(true)
    setLoginError('')

    try {
      await api.loginWeb(password)
      await mutateWebAuth() // Refresh auth status
      setShowPasswordLogin(false)
      setPassword('')
    } catch (error: any) {
      setLoginError(error.message || 'Login failed')
    } finally {
      setIsLoggingIn(false)
    }
  }

  // Show password login screen if required
  if (showPasswordLogin) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
        <div className="max-w-md w-full space-y-8">
          <div>
            <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
              Nanit Dashboard
            </h2>
            <p className="mt-2 text-center text-sm text-gray-600">
              Enter your password to access the dashboard
            </p>
          </div>
          <form className="mt-8 space-y-6" onSubmit={handlePasswordLogin}>
            <div>
              <label htmlFor="password" className="sr-only">
                Password
              </label>
              <input
                id="password"
                name="password"
                type="password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
                placeholder="Password"
              />
            </div>

            {loginError && (
              <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
                {loginError}
              </div>
            )}

            <div>
              <button
                type="submit"
                disabled={isLoggingIn}
                className="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
              >
                {isLoggingIn ? 'Signing in...' : 'Sign in'}
              </button>
            </div>
          </form>
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <MainLayout>
        <div className="flex items-center justify-center min-h-64">
          <LoadingSpinner size="lg" />
        </div>
      </MainLayout>
    )
  }

  if (isError) {
    return (
      <MainLayout>
        <ErrorMessage 
          title="Connection Error"
          message="Unable to connect to the Nanit bridge. Please check your connection."
        />
      </MainLayout>
    )
  }

  if (babies.length === 0) {
    return (
      <MainLayout>
        <div className="text-center py-12">
          <div className="card max-w-md mx-auto p-8">
            <h2 className="text-xl font-semibold text-nanit-gray-600 mb-2">
              No babies configured
            </h2>
            <p className="text-nanit-gray-500">
              Make sure you have authenticated and configured your Nanit account.
            </p>
          </div>
        </div>
      </MainLayout>
    )
  }

  return (
    <MainLayout>
      <div className="space-y-8">
        {babies.map((baby) => (
          <BabyCard key={baby.uid} baby={baby} />
        ))}
      </div>
    </MainLayout>
  )
}