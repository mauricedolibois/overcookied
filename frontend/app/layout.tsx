import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Overcookied",
  description: "A fun idle game built with Next.js and Go",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="antialiased">
        {children}
      </body>
    </html>
  );
}
