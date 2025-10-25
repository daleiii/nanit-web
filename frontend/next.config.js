/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export',
  trailingSlash: true,
  distDir: 'dist',
  images: {
    unoptimized: true
  },
  // Remove rewrites for static export - API calls will go directly to backend
  // assetPrefix: process.env.NODE_ENV === 'production' ? '/static' : '',
}

module.exports = nextConfig