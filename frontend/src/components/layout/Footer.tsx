export default function Footer() {
  return (
    <footer className="bg-white border-t border-nanit-gray-200 mt-12">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div className="text-center text-nanit-gray-500 space-y-3">
          <p className="text-sm font-medium">
            Nanit Dashboard
          </p>
          <div className="pt-3 border-t border-nanit-gray-200">
            <p className="text-xs">
              <strong>Disclaimer:</strong> This project has no association with Nanit. Much of the API reverse engineering came from{' '}
              <a 
                href="https://github.com/indiefan/home_assistant_nanit" 
                target="_blank" 
                rel="noopener noreferrer"
                className="text-blue-600 hover:text-blue-800 hover:underline"
              >
                this GitHub repository
              </a>
              .
            </p>
          </div>
        </div>
      </div>
    </footer>
  )
}