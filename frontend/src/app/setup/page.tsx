'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { api } from '@/lib/api'
import LoadingSpinner from '@/components/ui/LoadingSpinner'

export default function SetupPage() {
  const router = useRouter()
  const [step, setStep] = useState<'login' | '2fa'>('login')
  const [formData, setFormData] = useState({
    email: '',
    password: '',
    mfaCode: '',
  })
  const [mfaToken, setMfaToken] = useState<any>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    setError('')

    try {
      const response = await api.login(formData.email, formData.password)
      
      if (response.success && response.mfa_token) {
        setMfaToken(response.mfa_token)
        setStep('2fa')
      } else {
        setError(response.error || 'Login failed')
      }
    } catch (error) {
      setError('Failed to connect to server')
    } finally {
      setIsLoading(false)
    }
  }

  const handleVerify2FA = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    setError('')

    try {
      const response = await api.verify2FA(
        formData.email,
        formData.password,
        mfaToken,
        formData.mfaCode
      )
      
      if (response.success) {
        // Redirect to dashboard
        router.push('/')
      } else {
        setError(response.error || 'Verification failed')
      }
    } catch (error) {
      setError('Failed to verify code')
    } finally {
      setIsLoading(false)
    }
  }

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }))
    setError('')
  }

  return (
    <div className="min-h-screen bg-nanit-gray-50 flex items-center justify-center px-4">
      <div className="max-w-md w-full">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-gradient mb-2">
            Nanit Dashboard
          </h1>
          <p className="text-nanit-gray-600">
            Set up your Nanit Home Assistant Bridge
          </p>
        </div>

        <div className="card p-6">
          {step === 'login' ? (
            <>
              <h2 className="text-xl font-semibold text-nanit-gray-800 mb-6">
                Sign In to Nanit
              </h2>
              
              <form onSubmit={handleLogin} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-nanit-gray-700 mb-2">
                    Email
                  </label>
                  <input
                    type="email"
                    value={formData.email}
                    onChange={(e) => handleInputChange('email', e.target.value)}
                    className="w-full px-3 py-2 border border-nanit-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="your@email.com"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-nanit-gray-700 mb-2">
                    Password
                  </label>
                  <input
                    type="password"
                    value={formData.password}
                    onChange={(e) => handleInputChange('password', e.target.value)}
                    className="w-full px-3 py-2 border border-nanit-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="Your password"
                    required
                  />
                </div>

                {error && (
                  <div className="bg-red-50 border-l-4 border-red-500 p-3 rounded">
                    <div className="text-sm text-red-700">{error}</div>
                  </div>
                )}

                <button
                  type="submit"
                  disabled={isLoading}
                  className="w-full btn btn-primary disabled:opacity-50"
                >
                  {isLoading ? (
                    <div className="flex items-center justify-center gap-2">
                      <LoadingSpinner size="sm" />
                      <span>Signing In...</span>
                    </div>
                  ) : (
                    'Sign In'
                  )}
                </button>
              </form>
            </>
          ) : (
            <>
              <h2 className="text-xl font-semibold text-nanit-gray-800 mb-6">
                Two-Factor Authentication
              </h2>
              
              <div className="bg-blue-50 border-l-4 border-blue-500 p-3 rounded mb-6">
                <div className="text-sm text-blue-700">
                  Check your email for a verification code from Nanit and enter it below.
                </div>
              </div>

              <form onSubmit={handleVerify2FA} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-nanit-gray-700 mb-2">
                    Verification Code
                  </label>
                  <input
                    type="text"
                    value={formData.mfaCode}
                    onChange={(e) => handleInputChange('mfaCode', e.target.value)}
                    className="w-full px-3 py-2 border border-nanit-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="Enter 6-digit code"
                    maxLength={6}
                    required
                  />
                </div>

                {error && (
                  <div className="bg-red-50 border-l-4 border-red-500 p-3 rounded">
                    <div className="text-sm text-red-700">{error}</div>
                  </div>
                )}

                <div className="flex gap-3">
                  <button
                    type="button"
                    onClick={() => {
                      setStep('login')
                      setMfaToken(null)
                      setFormData(prev => ({ ...prev, mfaCode: '' }))
                    }}
                    className="flex-1 btn btn-secondary"
                  >
                    Back
                  </button>
                  
                  <button
                    type="submit"
                    disabled={isLoading}
                    className="flex-1 btn btn-primary disabled:opacity-50"
                  >
                    {isLoading ? (
                      <div className="flex items-center justify-center gap-2">
                        <LoadingSpinner size="sm" />
                        <span>Verifying...</span>
                      </div>
                    ) : (
                      'Verify'
                    )}
                  </button>
                </div>
              </form>
            </>
          )}
        </div>

        <div className="text-center mt-6 text-sm text-nanit-gray-500">
          <p>This will securely store your authentication for the Nanit bridge.</p>
        </div>
      </div>
    </div>
  )
}