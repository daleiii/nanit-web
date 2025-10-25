interface ErrorMessageProps {
  title?: string
  message: string
  action?: {
    label: string
    onClick: () => void
  }
}

export default function ErrorMessage({ title, message, action }: ErrorMessageProps) {
  return (
    <div className="card max-w-md mx-auto p-6 border-l-4 border-red-500">
      <div className="flex items-start">
        <div className="flex-shrink-0">
          <svg
            className="w-6 h-6 text-red-500"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4.5c-.77-.833-2.694-.833-3.464 0L3.34 16.5c-.77.833.192 2.5 1.732 2.5z"
            />
          </svg>
        </div>
        <div className="ml-3 flex-1">
          {title && (
            <h3 className="text-sm font-medium text-red-800 mb-1">
              {title}
            </h3>
          )}
          <p className="text-sm text-red-700">
            {message}
          </p>
          {action && (
            <button
              onClick={action.onClick}
              className="mt-3 text-sm font-medium text-red-800 hover:text-red-700 underline"
            >
              {action.label}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}