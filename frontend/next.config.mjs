/** @type {import('next').NextConfig} */
const nextConfig = {
  // Docker Compose serves the app with `next start` (Node server).
  // Cloudflare Pages needs fully static files: set STATIC_EXPORT=1 in the
  // Pages build environment to emit out/ instead. The game UI is fully
  // client-rendered, so static export works without changes.
  ...(process.env.STATIC_EXPORT === "1"
    ? { output: "export" }
    : { output: "standalone" }),
  reactStrictMode: true,
};

export default nextConfig;
