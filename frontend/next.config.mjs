/** @type {import('next').NextConfig} */
const nextConfig = {
  // Default is a static export (out/) served as Cloudflare Workers assets.
  // The Docker image sets NEXT_OUTPUT=standalone to run `node server.js`.
  output: process.env.NEXT_OUTPUT === "standalone" ? "standalone" : "export",
  // Static export has no image optimizer; fighter photos are already sized Wikimedia thumbnails.
  images: { unoptimized: true },
  reactStrictMode: true,
};

export default nextConfig;
