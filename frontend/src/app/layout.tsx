import type { Metadata } from "next";
import "./globals.css";
import { ClientProviders } from "./providers";
import { Sidebar } from "@/components/layout/sidebar";

export const metadata: Metadata = {
  title: "JobQueue — Background Job Management",
  description:
    "A modern dashboard for managing and monitoring background job queues. Track job status, create new jobs, and view statistics.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <ClientProviders>
          <Sidebar />
          <main className="lg:pl-56 min-h-screen">
            <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8 pt-16 lg:pt-8">
              {children}
            </div>
          </main>
        </ClientProviders>
      </body>
    </html>
  );
}
