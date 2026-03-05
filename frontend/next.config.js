/** @type {import('next').NextConfig} */
const nextConfig = {
    output: 'standalone',
    async rewrites() {
        return [
            {
                source: '/api/:path*',
                destination: `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/:path*`,
            },
        ]
    },
    experimental: {
        serverActions: { allowedOrigins: ['localhost:3000'] },
    },
}

module.exports = nextConfig
