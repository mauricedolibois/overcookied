export default function Home() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-amber-100 to-orange-200 dark:from-amber-900 dark:to-orange-950">
      <main className="flex flex-col items-center gap-8 p-8">
        <h1 className="text-6xl font-bold text-amber-900 dark:text-amber-100">
          🍪 Cookie Clicker
        </h1>
        <p className="text-xl text-amber-800 dark:text-amber-200 text-center max-w-md">
          Welcome to Cookie Clicker! A fun idle game built with Next.js and Go.
        </p>
        <div className="text-center">
          <p className="text-gray-700 dark:text-gray-300 mb-2">
            Frontend: Next.js {process.env.npm_package_dependencies_next || '16.0.3'}
          </p>
          <p className="text-gray-700 dark:text-gray-300">
            Backend: Go HTTP Server
          </p>
        </div>
      </main>
    </div>
  );
}
