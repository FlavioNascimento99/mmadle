import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "MMAdle — Daily MMA Fighter Guessing Game",
  description: "Guess the hidden UFC fighter of the day from six attribute clues.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
